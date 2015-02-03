// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package buildlet contains client tools for working with a buildlet
// server.
package buildlet // import "golang.org/x/build/buildlet"

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// NewClient returns a *Client that will manipulate ipPort,
// authenticated using the provided keypair.
//
// This constructor returns immediately without testing the host or auth.
func NewClient(ipPort string, kp KeyPair) *Client {
	return &Client{
		ipPort:   ipPort,
		tls:      kp,
		password: kp.Password(),
		httpClient: &http.Client{
			Transport: &http.Transport{
				Dial:    defaultDialer(),
				DialTLS: kp.tlsDialer(),
			},
		},
	}
}

// defaultDialer returns the net/http package's default Dial function.
// Notably, this sets TCP keep-alive values, so when we kill VMs
// (whose TCP stacks stop replying, forever), we don't leak file
// descriptors for otherwise forever-stalled TCP connections.
func defaultDialer() func(network, addr string) (net.Conn, error) {
	if fn := http.DefaultTransport.(*http.Transport).Dial; fn != nil {
		return fn
	}
	return net.Dial
}

// A Client interacts with a single buildlet.
type Client struct {
	ipPort     string
	tls        KeyPair
	password   string // basic auth password or empty for none
	httpClient *http.Client
}

// URL returns the buildlet's URL prefix, without a trailing slash.
func (c *Client) URL() string {
	if !c.tls.IsZero() {
		return "https://" + strings.TrimSuffix(c.ipPort, ":443")
	}
	return "http://" + strings.TrimSuffix(c.ipPort, ":80")
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	if c.password != "" {
		req.SetBasicAuth("gomote", c.password)
	}
	return c.httpClient.Do(req)
}

// doOK sends the request and expects a 200 OK response.
func (c *Client) doOK(req *http.Request) error {
	res, err := c.do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		slurp, _ := ioutil.ReadAll(io.LimitReader(res.Body, 4<<10))
		return fmt.Errorf("%v; body: %s", res.Status, slurp)
	}
	return nil
}

// PutTar writes files to the remote buildlet, rooted at the relative
// directory dir.
// If dir is empty, they're placed at the root of the buildlet's work directory.
// The dir is created if necessary.
// The Reader must be of a tar.gz file.
func (c *Client) PutTar(r io.Reader, dir string) error {
	req, err := http.NewRequest("PUT", c.URL()+"/writetgz?dir="+url.QueryEscape(dir), r)
	if err != nil {
		return err
	}
	return c.doOK(req)
}

// PutTarFromURL tells the buildlet to download the tar.gz file from tarURL
// and write it to dir, a relative directory from the workdir.
// If dir is empty, they're placed at the root of the buildlet's work directory.
// The dir is created if necessary.
// The url must be of a tar.gz file.
func (c *Client) PutTarFromURL(tarURL, dir string) error {
	form := url.Values{
		"url": {tarURL},
	}
	req, err := http.NewRequest("POST", c.URL()+"/writetgz?dir="+url.QueryEscape(dir), strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return c.doOK(req)
}

// GetTar returns a .tar.gz stream of the given directory, relative to the buildlet's work dir.
// The provided dir may be empty to get everything.
func (c *Client) GetTar(dir string) (tgz io.ReadCloser, err error) {
	req, err := http.NewRequest("GET", c.URL()+"/tgz?dir="+url.QueryEscape(dir), nil)
	if err != nil {
		return nil, err
	}
	res, err := c.do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		slurp, _ := ioutil.ReadAll(io.LimitReader(res.Body, 4<<10))
		res.Body.Close()
		return nil, fmt.Errorf("%v; body: %s", res.Status, slurp)
	}
	return res.Body, nil
}

// ExecOpts are options for a remote command invocation.
type ExecOpts struct {
	// Output is the output of stdout and stderr.
	// If nil, the output is discarded.
	Output io.Writer

	// Args are the arguments to pass to the cmd given to Client.Exec.
	Args []string

	// ExtraEnv are KEY=VALUE pairs to append to the buildlet
	// process's environment.
	ExtraEnv []string

	// SystemLevel controls whether the command is run outside of
	// the buildlet's environment.
	SystemLevel bool

	// OnStartExec is an optional hook that runs after the 200 OK
	// response from the buildlet, but before the output begins
	// writing to Output.
	OnStartExec func()
}

// Exec runs cmd on the buildlet.
//
// Two errors are returned: one is whether the command succeeded
// remotely (remoteErr), and the second (execErr) is whether there
// were system errors preventing the command from being started or
// seen to completition. If execErr is non-nil, the remoteErr is
// meaningless.
func (c *Client) Exec(cmd string, opts ExecOpts) (remoteErr, execErr error) {
	var mode string
	if opts.SystemLevel {
		mode = "sys"
	}
	form := url.Values{
		"cmd":    {cmd},
		"mode":   {mode},
		"cmdArg": opts.Args,
		"env":    opts.ExtraEnv,
	}
	req, err := http.NewRequest("POST", c.URL()+"/exec", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		slurp, _ := ioutil.ReadAll(io.LimitReader(res.Body, 4<<10))
		return nil, fmt.Errorf("buildlet: HTTP status %v: %s", res.Status, slurp)
	}
	condRun(opts.OnStartExec)

	// Stream the output:
	out := opts.Output
	if out == nil {
		out = ioutil.Discard
	}
	if _, err := io.Copy(out, res.Body); err != nil {
		return nil, fmt.Errorf("error copying response: %v", err)
	}

	// Don't record to the dashboard unless we heard the trailer from
	// the buildlet, otherwise it was probably some unrelated error
	// (like the VM being killed, or the buildlet crashing due to
	// e.g. https://golang.org/issue/9309, since we require a tip
	// build of the buildlet to get Trailers support)
	state := res.Trailer.Get("Process-State")
	if state == "" {
		return nil, errors.New("missing Process-State trailer from HTTP response; buildlet built with old (<= 1.4) Go?")
	}
	if state != "ok" {
		return errors.New(state), nil
	}
	return nil, nil
}

// Destroy shuts down the buildlet, destroying all state immediately.
func (c *Client) Destroy() error {
	req, err := http.NewRequest("POST", c.URL()+"/halt", nil)
	if err != nil {
		return err
	}
	return c.doOK(req)
}

// RemoveAll deletes the provided paths, relative to the work directory.
func (c *Client) RemoveAll(paths ...string) error {
	form := url.Values{"path": paths}
	req, err := http.NewRequest("POST", c.URL()+"/removeall", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	return c.doOK(req)
}

func condRun(fn func()) {
	if fn != nil {
		fn()
	}
}
