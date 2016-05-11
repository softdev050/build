// Copyright 2015 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Godash generates Go dashboards about issues and CLs.
//
// Usage:
//
//	godash [-cl] [-html]
//
// By default, godash prints a textual release dashboard to standard output.
// The release dashboard shows all open issues in the milestones for the upcoming
// release (currently Go 1.5), along with all open CLs mentioning those issues,
// and all other open CLs working in the main Go repository.
//
// If the -cl flag is specified, godash instead prints a CL dashboard, showing all
// open CLs, along with information about review status and review latency.
//
// If the -html flag is specified, godash prints HTML instead of text.
//
// Godash expects to find golang.org/x/build/cmd/cl and rsc.io/github/issue
// on its $PATH, to read data from Gerrit and GitHub.
//
// https://swtch.com/godash is periodically updated with the HTML versions of
// the two dashboards.
//
package main

import (
	"bytes"
	"crypto/md5"
	"flag"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/build/gerrit"
	"golang.org/x/build/godash"
)

const PointRelease = "Go1.6.1"
const Release = "Go1.7"

const (
	ProposalDir = "Pending Proposals"
	ClosedsDir  = "Closed Last Week"
)

var (
	output bytes.Buffer
	skipCL int

	days = flag.Int("days", 7, "number of days back")

	flagCL    = flag.Bool("cl", false, "print CLs only (no issues)")
	flagHTML  = flag.Bool("html", false, "print HTML output")
	flagMail  = flag.Bool("mail", false, "generate weekly mail")
	tokenFile = flag.String("token", "", "read GitHub token personal access token from `file` (default $HOME/.github-issue-token)")
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("godash: ")
	flag.Parse()
	if flag.NArg() != 0 {
		flag.Usage()
	}
	if *flagMail {
		*flagHTML = true
	}
	gh := godash.NewGitHubClient("golang/go", readAuthToken(), nil)
	ger := gerrit.NewClient("https://go-review.googlesource.com", gerrit.NoAuth)
	data := &godash.Data{}
	if err := data.FetchData(gh, ger, *days, *flagCL, *flagMail); err != nil {
		log.Fatalf("Failed to fetch data: %v", err)
	}

	if *flagMail {
		fmt.Fprintf(&output, "Go weekly status report\n")
	} else {
		what := "release"
		if *flagCL {
			what = "CL"
		}
		fmt.Fprintf(&output, "Go %s dashboard\n", what)
	}
	fmt.Fprintf(&output, "%v\n\n", time.Now().UTC().Format(time.UnixDate))
	if *flagHTML {
		fmt.Fprintf(&output, "HOWTO\n\n")
	}

	if *flagCL {
		data.PrintCLs(&output)
	} else {
		data.PrintIssues(&output)
	}

	if *flagHTML {
		printHTML()
		return
	}
	os.Stdout.Write(output.Bytes())
}

func readAuthToken() string {
	const short = ".github-issue-token"
	filename := filepath.Clean(os.Getenv("HOME") + "/" + short)
	shortFilename := filepath.Clean("$HOME/" + short)
	if *tokenFile != "" {
		filename = *tokenFile
		shortFilename = *tokenFile
	}
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal("reading token: ", err, "\n\n"+
			"Please create a personal access token at https://github.com/settings/tokens/new\n"+
			"and write it to ", shortFilename, " to use this program.\n"+
			"The token only needs the repo scope, or private_repo if you want to\n"+
			"view or edit issues for private repositories.\n"+
			"The benefit of using a personal access token over using your GitHub\n"+
			"password directly is that you can limit its use and revoke it at any time.\n\n")
	}
	fi, err := os.Stat(filename)
	if fi.Mode()&0077 != 0 {
		log.Fatalf("reading token: %s mode is %#o, want %#o", shortFilename, fi.Mode()&0777, fi.Mode()&0700)
	}
	return strings.TrimSpace(string(data))
}

func printHTML() {
	data := html.EscapeString(output.String())
	i := strings.Index(data, "\n")
	if i < 0 {
		i = len(data)
	}
	if *flagMail {
		fmt.Printf("Subject: Go weekly report for %s\n", time.Now().Format("2006-01-02"))
		fmt.Printf("From: \"Gopher Robot\" <gobot@golang.org>\n")
		fmt.Printf("To: golang-dev@googlegroups.com\n")
		fmt.Printf("Message-Id: <godash.%x@golang.org>\n", md5.Sum([]byte(data)))
		fmt.Printf("Content-Type: text/html; charset=utf-8\n")
		fmt.Printf("\n")
	}
	fmt.Printf("<html>\n")
	fmt.Printf("<meta charset=\"UTF-8\">\n")
	fmt.Printf("<title>%s</title>\n", data[:i])
	fmt.Printf("<style>\n")
	fmt.Printf(".early {}\n")
	fmt.Printf(".maybe {}\n")
	fmt.Printf(".late {color: #700; text-decoration: underline;}\n")
	fmt.Printf(".closed {background-color: #eee;}\n")
	fmt.Printf("hr {border: none; border-top: 2px solid #000; height: 5px; border-bottom: 1px solid #000;}\n")
	fmt.Printf("</style>\n")
	fmt.Printf("<pre>\n")
	data = regexp.MustCompile(`(?m)^HOWTO`).ReplaceAllString(data, `<a target="_blank" href="index.html">about the dashboard</a>`)
	data = regexp.MustCompile(`(CL (\d+))\b`).ReplaceAllString(data, "<a target=\"_blank\" href='https://golang.org/cl/$2'>$1</a>")
	data = regexp.MustCompile(`(#(\d\d\d+))\b`).ReplaceAllString(data, "<a target=\"_blank\" href='https://golang.org/issue/$2'>$1</a>")
	data = regexp.MustCompile(`(?m)^(Closed Last Week|Pending Proposals|Pending CLs|Go[\?A-Za-z0-9][^\n]*)`).ReplaceAllString(data, "<hr><b><font size='+1'>$1</font></b>")
	data = regexp.MustCompile(`(?m)^([\?A-Za-z0-9][^\n]*)`).ReplaceAllString(data, "<b>$1</b>")
	data = regexp.MustCompile(`(?m)^([^\n]*\[early[^\n]*)`).ReplaceAllString(data, "<span class='early'>$1</span>")
	data = regexp.MustCompile(`(?m)^([^\n]*\[maybe[^\n]*)`).ReplaceAllString(data, "<span class='maybe'>$1</span>")
	data = regexp.MustCompile(`(?m)^( +)(.*)( → )(.*)(, [\d/]+ days)(, waiting for reviewer)`).ReplaceAllString(data, "$1$2$3<b>$4</b>$5$6")
	data = regexp.MustCompile(`(?m)^( +)(.*)( → )(.*)(, [\d/]+ days)(, waiting for author)`).ReplaceAllString(data, "$1<b>$2</b>$3$4$5$6")
	data = regexp.MustCompile(`(→ )(.*, \d\d+)(/\d+ days)(, waiting for reviewer)`).ReplaceAllString(data, "$1<b class='late'>$2</b>$3$4")
	fmt.Printf("%s\n", data)
	fmt.Printf("</pre>\n")
}
