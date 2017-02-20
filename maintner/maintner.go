// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package maintner mirrors, searches, syncs, and serves Git, Github,
// and Gerrit metadata.
//
// Maintner is short for "Maintainer". This package is intended for
// use by many tools. The name of the daemon that serves the maintner
// data to other tools is "maintnerd".
package maintner

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/github"
	"golang.org/x/build/maintner/maintpb"
	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"
)

// Corpus holds all of a project's metadata.
//
// There are two main phases to the Corpus: the catch-up phase, when the Corpus
// is populated from a MutationSource (disk, database), and the polling phase,
// when the Corpus polls for new events and stores/writes them to disk. Call
// StartLogging between the catch-up phase and the polling phase.
type Corpus struct {
	// ... TODO

	mu           sync.RWMutex
	githubIssues map[githubRepo]map[int32]*githubIssue // repo -> num -> issue
	githubUsers  map[int64]*githubUser
	githubRepos  []repoObj
	// If true, log new commits
	shouldLog bool
}

type repoObj struct {
	name      githubRepo
	tokenFile string
}

func NewCorpus() *Corpus {
	return &Corpus{
		githubIssues: make(map[githubRepo]map[int32]*githubIssue),
		githubUsers:  make(map[int64]*githubUser),
		githubRepos:  []repoObj{},
	}
}

// StartLogging indicates that further changes should be written to the log.
func (c *Corpus) StartLogging() {
	c.mu.Lock()
	c.shouldLog = true
	c.mu.Unlock()
}

func (c *Corpus) AddGithub(owner, repo, tokenFile string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.githubRepos = append(c.githubRepos, repoObj{
		name:      githubRepo(owner + "/" + repo),
		tokenFile: tokenFile,
	})
}

// githubRepo is a github org & repo, lowercase, joined by a '/',
// such as "golang/go".
type githubRepo string

// githubUser represents a github user.
// It is a subset of https://developer.github.com/v3/users/#get-a-single-user
type githubUser struct {
	ID    int64
	Login string
}

// githubIssue represents a github issue.
// See https://developer.github.com/v3/issues/#get-a-single-issue
type githubIssue struct {
	ID      int64
	Number  int32
	Closed  bool
	User    *githubUser
	Created time.Time
	Updated time.Time
	Body    string
	// TODO Comments ...
}

// A MutationSource yields a log of mutations that will catch a corpus
// back up to the present.
type MutationSource interface {
	// GetMutations returns a channel of mutations.
	// The channel should be closed at the end.
	// All sends on the returned channel should select
	// on the provided context.
	GetMutations(context.Context) <-chan *maintpb.Mutation
}

// Initialize populates the Corpus using the data from the MutationSource.
func (c *Corpus) Initialize(ctx context.Context, src MutationSource) error {
	return c.processMutations(ctx, src)
}

func (c *Corpus) processMutations(ctx context.Context, src MutationSource) error {
	ch := src.GetMutations(ctx)
	done := ctx.Done()

	c.mu.Lock()
	defer c.mu.Unlock()
	for {
		select {
		case <-done:
			return ctx.Err()
		case m, ok := <-ch:
			if !ok {
				return nil
			}
			c.processMutationLocked(m)
		}
	}
}

// c.mu must be held.
func (c *Corpus) processMutationLocked(m *maintpb.Mutation) {
	if im := m.GithubIssue; im != nil {
		c.processGithubIssueMutation(im)
	}
	// TODO: more...
}

func (c *Corpus) repoKey(owner, repo string) githubRepo {
	if owner == "" || repo == "" {
		return ""
	}
	// TODO: avoid garbage, use interned strings? profile later
	// once we have gigabytes of mutation logs to slurp at
	// start-up. (The same thing mattered for Camlistore start-up
	// time at least)
	return githubRepo(owner + "/" + repo)
}

func (c *Corpus) getGithubUser(pu *maintpb.GithubUser) *githubUser {
	if pu == nil {
		return nil
	}
	if u := c.githubUsers[pu.Id]; u != nil {
		if pu.Login != "" && pu.Login != u.Login {
			u.Login = pu.Login
		}
		return u
	}
	if c.githubUsers == nil {
		c.githubUsers = make(map[int64]*githubUser)
	}
	u := &githubUser{
		ID:    pu.Id,
		Login: pu.Login,
	}
	c.githubUsers[pu.Id] = u
	return u
}

func (c *Corpus) processGithubIssueMutation(m *maintpb.GithubIssueMutation) {
	k := c.repoKey(m.Owner, m.Repo)
	if k == "" {
		// TODO: errors? return false? skip for now.
		return
	}
	if m.Number == 0 {
		return
	}
	issueMap, ok := c.githubIssues[k]
	if !ok {
		if c.githubIssues == nil {
			c.githubIssues = make(map[githubRepo]map[int32]*githubIssue)
		}
		issueMap = make(map[int32]*githubIssue)
		c.githubIssues[k] = issueMap
	}
	gi, ok := issueMap[m.Number]
	if !ok {
		gi = &githubIssue{
			Number: m.Number,
			User:   c.getGithubUser(m.User),
		}
		issueMap[m.Number] = gi
	}
	if m.Body != "" {
		gi.Body = m.Body
	}
	// TODO: times, etc.
}

// PopulateFromServer populates the corpus from a maintnerd server.
func (c *Corpus) PopulateFromServer(ctx context.Context, serverURL string) error {
	panic("TODO")
}

// PopulateFromDisk populates the corpus from a set of mutation logs
// in a local directory.
func (c *Corpus) PopulateFromDisk(ctx context.Context, dir string) error {
	panic("TODO")
}

// PopulateFromAPIs populates the corpus using API calls to
// the upstream Git, Github, and/or Gerrit servers.
func (c *Corpus) PopulateFromAPIs(ctx context.Context) error {
	panic("TODO")
}

// Poll checks for new changes on all repositories being tracked by the Corpus.
func (c *Corpus) Poll(ctx context.Context) error {
	group, ctx := errgroup.WithContext(ctx)
	for _, rp := range c.githubRepos {
		rp := rp
		group.Go(func() error {
			return c.PollGithubLoop(ctx, rp.name, rp.tokenFile)
		})
	}
	return group.Wait()
}

// PollGithubLoop checks for new changes on a single Github repository and
// updates the Corpus with any changes.
func (c *Corpus) PollGithubLoop(ctx context.Context, rp githubRepo, tokenFile string) error {
	slurp, err := ioutil.ReadFile(tokenFile)
	if err != nil {
		return err
	}
	f := strings.SplitN(strings.TrimSpace(string(slurp)), ":", 2)
	if len(f) != 2 || f[0] == "" || f[1] == "" {
		return fmt.Errorf("Expected token file %s to be of form <username>:<token>", tokenFile)
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: f[1]})
	tc := oauth2.NewClient(oauth2.NoContext, ts)
	ghc := github.NewClient(tc)
	for {
		err := c.pollGithub(ctx, rp, ghc)
		if err == context.Canceled {
			return err
		}
		log.Printf("Polled github for %s; err = %v. Sleeping.", rp, err)
		time.Sleep(30 * time.Second)
	}
}

func (c *Corpus) pollGithub(ctx context.Context, rp githubRepo, ghc *github.Client) error {
	log.Printf("Polling github for %s ...", rp)
	page := 1
	keepGoing := true
	splits := strings.SplitN(string(rp), "/", 2)
	owner, repo := splits[0], splits[1]
	for keepGoing {
		// TODO: use https://godoc.org/github.com/google/go-github/github#ActivityService.ListIssueEventsForRepository probably
		issues, res, err := ghc.Issues.ListByRepo(context.TODO(), owner, repo, &github.IssueListByRepoOptions{
			State:     "all",
			Sort:      "updated",
			Direction: "desc",
			ListOptions: github.ListOptions{
				Page:    page,
				PerPage: 100,
			},
		})
		if err != nil {
			return err
		}
		log.Printf("github %s/%s: page %d, num issues %d, res: %#v", owner, repo, page, len(issues), res)
		keepGoing = false
		for _, is := range issues {
			fmt.Printf("issue %d: %s\n", is.ID, *is.Title)
			changes := false
			if changes {
				keepGoing = true
			}
		}
		page++
	}
	return nil
}
