// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime"
	"strings"
	"text/template"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/delay"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
)

const (
	mailFrom   = "builder@golang.org" // use this for sending any mail
	failMailTo = "golang-dev@googlegroups.com"
	domain     = "build.golang.org"
	gobotBase  = "http://research.swtch.com/gobot_codereview"
)

// ignoreFailure is a set of builders that we don't email about because
// they are not yet production-ready.
var ignoreFailure = map[string]bool{
	"dragonfly-386":         true,
	"dragonfly-amd64":       true,
	"freebsd-arm":           true,
	"netbsd-amd64-bsiegert": true,
	"netbsd-arm-rpi":        true,
	"plan9-amd64-aram":      true,
}

// notifyOnFailure checks whether the supplied Commit or the subsequent
// Commit (if present) breaks the build for this builder.
// If either of those commits break the build an email notification is sent
// from a delayed task. (We use a task because this way the mail won't be
// sent if the enclosing datastore transaction fails.)
//
// This must be run in a datastore transaction, and the provided *Commit must
// have been retrieved from the datastore within that transaction.
func notifyOnFailure(c context.Context, com *Commit, builder string) error {
	if ignoreFailure[builder] {
		return nil
	}

	// TODO(adg): implement notifications for packages
	if com.PackagePath != "" {
		return nil
	}

	p := &Package{Path: com.PackagePath}
	var broken *Commit
	cr := com.Result(builder, "")
	if cr == nil {
		return fmt.Errorf("no result for %s/%s", com.Hash, builder)
	}
	q := datastore.NewQuery("Commit").Ancestor(p.Key(c))
	if cr.OK {
		// This commit is OK. Notify if next Commit is broken.
		next := new(Commit)
		q = q.Filter("ParentHash=", com.Hash)
		if err := firstMatch(c, q, next); err != nil {
			if err == datastore.ErrNoSuchEntity {
				// OK at tip, no notification necessary.
				return nil
			}
			return err
		}
		if nr := next.Result(builder, ""); nr != nil && !nr.OK {
			log.Debugf(c, "commit ok: %#v\nresult: %#v", com, cr)
			log.Debugf(c, "next commit broken: %#v\nnext result:%#v", next, nr)
			broken = next
		}
	} else {
		// This commit is broken. Notify if the previous Commit is OK.
		prev := new(Commit)
		q = q.Filter("Hash=", com.ParentHash)
		if err := firstMatch(c, q, prev); err != nil {
			if err == datastore.ErrNoSuchEntity {
				// No previous result, let the backfill of
				// this result trigger the notification.
				return nil
			}
			return err
		}
		if pr := prev.Result(builder, ""); pr != nil && pr.OK {
			log.Debugf(c, "commit broken: %#v\nresult: %#v", com, cr)
			log.Debugf(c, "previous commit ok: %#v\nprevious result:%#v", prev, pr)
			broken = com
		}
	}
	if broken == nil {
		return nil
	}
	r := broken.Result(builder, "")
	if r == nil {
		return fmt.Errorf("finding result for %q: %+v", builder, com)
	}
	return commonNotify(c, broken, builder, r.LogHash)
}

// firstMatch executes the query q and loads the first entity into v.
func firstMatch(c context.Context, q *datastore.Query, v interface{}) error {
	t := q.Limit(1).Run(c)
	_, err := t.Next(v)
	if err == datastore.Done {
		err = datastore.ErrNoSuchEntity
	}
	return err
}

var (
	notifyLater = delay.Func("notify", notify)
	notifyTmpl  = template.Must(template.New("notify.txt").
			Funcs(template.FuncMap(tmplFuncs)).ParseFiles(templateFile("notify.txt")))
)

// notify tries to update the CL for the given Commit with a failure message.
// If it doesn't succeed, it sends a failure email to golang-dev.
func notify(c context.Context, com *Commit, builder, logHash string) {
	var msg bytes.Buffer
	err := notifyTmpl.Execute(&msg, struct {
		Builder  string
		LogHash  string
		Hostname string
	}{builder, logHash, domain})
	if err != nil {
		log.Criticalf(c, "couldn't render template: %v", err)
		return
	}
	if err := postGerritMessage(c, com, msg.String()); err != nil {
		log.Errorf(c, "couldn't post to gerrit: %v", err)
	}
}

// postGerritMessage posts a message to the code review thread for the given
// Commit.
func postGerritMessage(c context.Context, com *Commit, message string) error {
	if appengine.IsDevAppServer() {
		log.Infof(c, "Skiping update of Gerrit review for %v with message: %v", com, message)
		return nil
	}
	// Get change ID using commit hash.
	resp, err := urlfetch.Client(c).Get("https://go-review.googlesource.com/r/" + com.Hash)
	if err != nil {
		return fmt.Errorf("lookup %v: contacting Gerrit %v", com.Hash, err)
	}
	resp.Body.Close()
	if resp.Request == nil || resp.Request.URL == nil {
		return fmt.Errorf("lookup %s: missing request info in http response", com.Hash)
	}
	frag := resp.Request.URL.Fragment
	if !strings.HasPrefix(frag, "/c/") || !strings.HasSuffix(frag, "/") {
		return fmt.Errorf("lookup %s: unexpected URL fragment: #%s", com.Hash, frag)
	}
	id := frag[len("/c/") : len(frag)-len("/")]

	// Prepare request.
	msg := struct {
		Message string `json:"message"`
	}{message}
	data, err := json.Marshal(&msg)
	if err != nil {
		return fmt.Errorf("marshalling message: %v", err)
	}
	req, err := http.NewRequest("POST", "https://go-review.googlesource.com/a/changes/"+id+"/revisions/current/review", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("preparing message: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(Config(c, "GerritUsername"), Config(c, "GerritPassword"))

	// Make request.
	resp, err = urlfetch.Client(c).Do(req)
	if err != nil {
		return fmt.Errorf("posting message: %v", err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf("reading response body: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("posting message: %s\n%s", resp.Status, body)
	}
	return nil
}

// commonNotify MUST!!! be called from within a transaction inside which
// the provided Commit entity was retrieved from the datastore.
func commonNotify(c context.Context, com *Commit, builder, logHash string) error {
	if com.Num == 0 || com.Desc == "" {
		stk := make([]byte, 10000)
		n := runtime.Stack(stk, false)
		stk = stk[:n]
		log.Errorf(c, "refusing to notify with com=%+v\n%s", *com, string(stk))
		return fmt.Errorf("misuse of commonNotify")
	}
	if com.FailNotificationSent {
		return nil
	}
	log.Infof(c, "%s is broken commit; notifying", com.Hash)
	notifyLater.Call(c, com, builder, logHash) // add task to queue
	com.FailNotificationSent = true
	return putCommit(c, com)
}
