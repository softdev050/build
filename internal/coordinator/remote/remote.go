// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package remote

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"golang.org/x/build/buildlet"
	"golang.org/x/build/internal"
)

const (
	remoteBuildletIdleTimeout   = 30 * time.Minute
	remoteBuildletCleanInterval = time.Minute
)

// Session stores the metadata for a remote buildlet Session.
type Session struct {
	BuilderType string // default builder config to use if not overwritten
	Expires     time.Time
	HostType    string
	ID          string // unique identifier for instance "user-bradfitz-linux-amd64-0"
	OwnerID     string // identity aware proxy user id: "accounts.google.com:userIDvalue"
	buildlet    buildlet.Client
	created     time.Time
}

// renew extends the expiration timestamp for a session.
// The SessionPool lock should be held before calling.
func (s *Session) renew() {
	s.Expires = time.Now().Add(remoteBuildletIdleTimeout)
}

// isExpired determines if the remote buildlet session has expired.
// The SessionPool lock should be held before calling.
func (s *Session) isExpired() bool {
	return !s.Expires.IsZero() && s.Expires.Before(time.Now())
}

// SessionPool contains active remote buildlet sessions.
type SessionPool struct {
	mu sync.RWMutex

	once       sync.Once
	pollWait   sync.WaitGroup
	cancelPoll context.CancelFunc
	m          map[string]*Session // keyed by buildletName
}

// NewSessionPool creates a session pool which stores and provides access to active remote buildlet sessions.
// Either cancelling the context or calling close on the session pool will terminate any polling functions.
func NewSessionPool(ctx context.Context) *SessionPool {
	ctx, cancel := context.WithCancel(ctx)
	sp := &SessionPool{
		cancelPoll: cancel,
		m:          map[string]*Session{},
	}
	sp.pollWait.Add(1)
	go func() {
		internal.PeriodicallyDo(ctx, remoteBuildletCleanInterval, func(ctx context.Context, _ time.Time) {
			log.Printf("remote: cleaning up expired remote buildlets")
			sp.destroyExpiredSessions(ctx)
		})
		sp.pollWait.Done()
	}()
	return sp
}

// AddSession adds the provided session to the session pool.
func (sp *SessionPool) AddSession(ownerID, username, builderType, hostType string, bc buildlet.Client) (name string) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	for n := 0; ; n++ {
		name = fmt.Sprintf("%s-%s-%d", username, builderType, n)
		if _, ok := sp.m[name]; !ok {
			now := time.Now()
			sp.m[name] = &Session{
				BuilderType: builderType,
				buildlet:    bc,
				created:     now,
				Expires:     now.Add(remoteBuildletIdleTimeout),
				HostType:    hostType,
				ID:          name,
				OwnerID:     ownerID,
			}
			return name
		}
	}
}

// IsGCESession checks if the session is a GCE instance.
func (sp *SessionPool) IsGCESession(instName string) bool {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	for _, s := range sp.m {
		if s.buildlet.GCEInstanceName() == instName {
			return true
		}
	}
	return false
}

// destroyExpiredSessions destroys all sessions which have expired.
func (sp *SessionPool) destroyExpiredSessions(ctx context.Context) {
	sp.mu.Lock()
	var ss []*Session
	for name, s := range sp.m {
		if s.isExpired() {
			ss = append(ss, s)
			delete(sp.m, name)
		}
	}
	sp.mu.Unlock()
	// the sessions are no longer in the map. They can be mutated.
	for _, s := range ss {
		if err := s.buildlet.Close(); err != nil {
			log.Printf("remote: unable to close buildlet connection %s", err)
		}
	}
}

// DestroySession destroys a session.
func (sp *SessionPool) DestroySession(buildletName string) error {
	sp.mu.Lock()
	s, ok := sp.m[buildletName]
	if ok {
		delete(sp.m, buildletName)
	}
	sp.mu.Unlock()
	if !ok {
		return fmt.Errorf("remote buildlet does not exist=%s", buildletName)
	}
	if err := s.buildlet.Close(); err != nil {
		log.Printf("remote: unable to close buildlet connection %s: %s", buildletName, err)
	}
	return nil
}

// Close cancels the polling performed by the session pool. It waits for polling to conclude
// before returning.
func (sp *SessionPool) Close() {
	sp.once.Do(func() {
		sp.cancelPoll()
		sp.pollWait.Wait()
	})
}

// List returns a list of all active sessions sorted by session ID.
func (sp *SessionPool) List() []*Session {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	var ss []*Session
	for _, s := range sp.m {
		ss = append(ss, &Session{
			BuilderType: s.BuilderType,
			Expires:     s.Expires,
			HostType:    s.HostType,
			ID:          s.ID,
			OwnerID:     s.OwnerID,
		})
	}
	sort.Slice(ss, func(i, j int) bool { return ss[i].ID < ss[j].ID })
	return ss
}

// Len gives a count of how many sessions are in the pool.
func (sp *SessionPool) Len() int {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	return len(sp.m)
}

// Session retrieves information about the instance associated with a session from the pool.
func (sp *SessionPool) Session(buildletName string) (*Session, error) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if s, ok := sp.m[buildletName]; ok {
		s.renew()
		return &Session{
			BuilderType: s.BuilderType,
			Expires:     s.Expires,
			HostType:    s.HostType,
			ID:          s.ID,
			OwnerID:     s.OwnerID,
		}, nil
	}
	return nil, fmt.Errorf("remote buildlet does not exist=%s", buildletName)
}

// Buildlet returns the buildlet client associated with the Session.
func (sp *SessionPool) BuildletClient(buildletName string) (buildlet.Client, error) {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	s, ok := sp.m[buildletName]
	if !ok {
		return nil, fmt.Errorf("remote buildlet does not exist=%s", buildletName)
	}
	return s.buildlet, nil
}

// KeepAlive will renew the remote buildlet session by extending the expiration value. It will
// periodically extend the value until the provided context has been cancelled.
func (sp *SessionPool) KeepAlive(ctx context.Context, buildletName string) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	s, ok := sp.m[buildletName]
	if !ok {
		return fmt.Errorf("remote buildlet does not exist=%s", buildletName)
	}
	go internal.PeriodicallyDo(ctx, time.Minute, func(ctx context.Context, _ time.Time) {
		sp.mu.Lock()
		s.renew()
		sp.mu.Unlock()
	})
	return nil
}
