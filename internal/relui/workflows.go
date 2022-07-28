// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package relui

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"math/rand"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v4/pgxpool"
	"golang.org/x/build/buildlet"
	"golang.org/x/build/dashboard"
	"golang.org/x/build/internal/gcsfs"
	"golang.org/x/build/internal/releasetargets"
	"golang.org/x/build/internal/relui/db"
	"golang.org/x/build/internal/task"
	wf "golang.org/x/build/internal/workflow"
	"golang.org/x/net/context/ctxhttp"
)

// DefinitionHolder holds workflow definitions.
type DefinitionHolder struct {
	mu          sync.Mutex
	definitions map[string]*wf.Definition
}

// NewDefinitionHolder creates a new DefinitionHolder,
// initialized with a sample "echo" wf.
func NewDefinitionHolder() *DefinitionHolder {
	return &DefinitionHolder{definitions: map[string]*wf.Definition{
		"echo": newEchoWorkflow(),
	}}
}

// Definition returns the initialized wf.Definition registered
// for a given name.
func (h *DefinitionHolder) Definition(name string) *wf.Definition {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.definitions[name]
}

// RegisterDefinition registers a definition with a name.
// If a definition with the same name already exists, RegisterDefinition panics.
func (h *DefinitionHolder) RegisterDefinition(name string, d *wf.Definition) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, exist := h.definitions[name]; exist {
		panic("relui: multiple registrations for " + name)
	}
	h.definitions[name] = d
}

// Definitions returns the names of all registered definitions.
func (h *DefinitionHolder) Definitions() map[string]*wf.Definition {
	h.mu.Lock()
	defer h.mu.Unlock()
	defs := make(map[string]*wf.Definition)
	for k, v := range h.definitions {
		defs[k] = v
	}
	return defs
}

// RegisterMailDLCLDefinition registers a workflow definition for mailing a golang.org/dl CL
// onto h.
//
// This is superseded by RegisterReleaseWorkflows and will be removed
// after some time, when we confirm there's no need for separate workflows.
func RegisterMailDLCLDefinition(h *DefinitionHolder, tasks *task.VersionTasks) {
	versions := wf.ParamDef[[]string]{
		Name:      "Versions",
		ParamType: wf.SliceShort,
		Doc: `Versions are the Go versions that have been released.

The versions must use the same format as Go tags,
and the list must contain one or two versions.

For example:
• "go1.18.2" and "go1.17.10" for a minor Go release
• "go1.19" for a major Go release
• "go1.19beta1" or "go1.19rc1" for a pre-release`,
	}

	wd := wf.New()
	wf.Output(wd, "ChangeURL", wf.Task1(wd, "mail-dl-cl", func(ctx *wf.TaskContext, versions []string) (string, error) {
		id, err := tasks.MailDLCL(ctx, versions, false)
		if err != nil {
			return "", err
		}
		return task.ChangeLink(id), nil
	}, wf.Param(wd, versions)))
	h.RegisterDefinition("mail-dl-cl", wd)
}

// RegisterCommunicationDefinitions registers workflow definitions
// involving mailing announcements and posting tweets onto h.
//
// This is superseded by RegisterReleaseWorkflows and will be removed
// after some time, when we confirm there's no need for separate workflows.
func RegisterCommunicationDefinitions(h *DefinitionHolder, tasks task.CommunicationTasks) {
	version := wf.ParamDef[string]{
		Name: "Version",
		Doc: `Version is the Go version that has been released.

The version string must use the same format as Go tags.`,
	}

	{
		wd := wf.New()

		minorVersion := version
		minorVersion.Example = "go1.18.2"
		v1 := wf.Param(wd, minorVersion)
		v2 := wf.Param(wd, wf.ParamDef[string]{
			Name: "Secondary Version (optional)",
			Doc: `Secondary Version is an older Go version that was also released,
or the empty string if only one minor release was made.`,
			Example: "go1.17.10",
		})
		securitySummary := wf.Param(wd, securitySummaryParameter)
		securityFixes := wf.Param(wd, securityFixesParameter)
		names := wf.Param(wd, releaseCoordinatorNames)

		sentMail := wf.Task4(wd, "mail-announcement", func(ctx *wf.TaskContext, v1, v2 string, sec, names []string) (task.SentMail, error) {
			return tasks.AnnounceRelease(ctx, task.ReleaseAnnouncement{Version: v1, SecondaryVersion: v2, Security: sec, Names: names})
		}, v1, v2, securityFixes, names)
		announcementURL := wf.Task1(wd, "await-announcement", tasks.AwaitAnnounceMail, sentMail)
		tweetURL := wf.Task4(wd, "post-tweet", func(ctx *wf.TaskContext, v1, v2, sec, ann string) (string, error) {
			return tasks.TweetRelease(ctx, task.ReleaseTweet{Version: v1, SecondaryVersion: v2, Security: sec, Announcement: ann})
		}, v1, v2, securitySummary, announcementURL)

		wf.Output(wd, "AnnouncementURL", announcementURL)
		wf.Output(wd, "TweetURL", tweetURL)

		h.RegisterDefinition("announce-and-tweet-minor", wd)
	}
	{
		wd := wf.New()

		betaVersion := version
		betaVersion.Example = "go1.19beta1"
		v := wf.Param(wd, betaVersion)
		names := wf.Param(wd, releaseCoordinatorNames)

		sentMail := wf.Task2(wd, "mail-announcement", func(ctx *wf.TaskContext, v string, names []string) (task.SentMail, error) {
			return tasks.AnnounceRelease(ctx, task.ReleaseAnnouncement{Version: v, Names: names})
		}, v, names)
		announcementURL := wf.Task1(wd, "await-announcement", tasks.AwaitAnnounceMail, sentMail)
		tweetURL := wf.Task2(wd, "post-tweet", func(ctx *wf.TaskContext, v, ann string) (string, error) {
			return tasks.TweetRelease(ctx, task.ReleaseTweet{Version: v, Announcement: ann})
		}, v, announcementURL)

		wf.Output(wd, "AnnouncementURL", announcementURL)
		wf.Output(wd, "TweetURL", tweetURL)

		h.RegisterDefinition("announce-and-tweet-beta", wd)
	}
	{
		wd := wf.New()

		rcVersion := version
		rcVersion.Example = "go1.19rc1"
		v := wf.Param(wd, rcVersion)
		names := wf.Param(wd, releaseCoordinatorNames)

		sentMail := wf.Task2(wd, "mail-announcement", func(ctx *wf.TaskContext, v string, names []string) (task.SentMail, error) {
			return tasks.AnnounceRelease(ctx, task.ReleaseAnnouncement{Version: v, Names: names})
		}, v, names)
		announcementURL := wf.Task1(wd, "await-announcement", tasks.AwaitAnnounceMail, sentMail)
		tweetURL := wf.Task2(wd, "post-tweet", func(ctx *wf.TaskContext, v, ann string) (string, error) {
			return tasks.TweetRelease(ctx, task.ReleaseTweet{Version: v, Announcement: ann})
		}, v, announcementURL)

		wf.Output(wd, "AnnouncementURL", announcementURL)
		wf.Output(wd, "TweetURL", tweetURL)

		h.RegisterDefinition("announce-and-tweet-rc", wd)
	}
	{
		wd := wf.New()

		majorVersion := version
		majorVersion.Example = "go1.19"
		v := wf.Param(wd, majorVersion)
		names := wf.Param(wd, releaseCoordinatorNames)

		sentMail := wf.Task2(wd, "mail-announcement", func(ctx *wf.TaskContext, v string, names []string) (task.SentMail, error) {
			return tasks.AnnounceRelease(ctx, task.ReleaseAnnouncement{Version: v, Names: names})
		}, v, names)
		announcementURL := wf.Task1(wd, "await-announcement", tasks.AwaitAnnounceMail, sentMail)
		tweetURL := wf.Task1(wd, "post-tweet", func(ctx *wf.TaskContext, v string) (string, error) {
			return tasks.TweetRelease(ctx, task.ReleaseTweet{Version: v})
		}, v)

		wf.Output(wd, "AnnouncementURL", announcementURL)
		wf.Output(wd, "TweetURL", tweetURL)

		h.RegisterDefinition("announce-and-tweet-major", wd)
	}
}

// Release parameter definitions.
var (
	securitySummaryParameter = wf.ParamDef[string]{
		Name: "Security Summary (optional)",
		Doc: `Security Summary is an optional sentence describing security fixes included in this release.

It shows up in the release tweet.

The empty string means there are no security fixes to highlight.

Past examples:
• "Includes a security fix for crypto/tls (CVE-2021-34558)."
• "Includes a security fix for the Wasm port (CVE-2021-38297)."
• "Includes security fixes for encoding/pem (CVE-2022-24675), crypto/elliptic (CVE-2022-28327), crypto/x509 (CVE-2022-27536)."`,
	}

	securityFixesParameter = wf.ParamDef[[]string]{
		Name:      "Security Fixes (optional)",
		ParamType: wf.SliceLong,
		Doc: `Security Fixes is a list of descriptions, one for each distinct security fix included in this release, in Markdown format.

It shows up in the announcement mail.

The empty list means there are no security fixes included.

Past examples:
• "encoding/pem: fix stack overflow in Decode

   A large (more than 5 MB) PEM input can cause a stack overflow in Decode,
   leading the program to crash.

   Thanks to Juho Nurminen of Mattermost who reported the error.

   This is CVE-2022-24675 and Go issue https://go.dev/issue/51853."
• "crypto/elliptic: tolerate all oversized scalars in generic P-256

   A crafted scalar input longer than 32 bytes can cause P256().ScalarMult
   or P256().ScalarBaseMult to panic. Indirect uses through crypto/ecdsa and
   crypto/tls are unaffected. amd64, arm64, ppc64le, and s390x are unaffected.

   This was discovered thanks to a Project Wycheproof test vector.

   This is CVE-2022-28327 and Go issue https://go.dev/issue/52075."`,
		Example: `encoding/pem: fix stack overflow in Decode

A large (more than 5 MB) PEM input can cause a stack overflow in Decode,
leading the program to crash.

Thanks to Juho Nurminen of Mattermost who reported the error.

This is CVE-2022-24675 and Go issue https://go.dev/issue/51853.`,
	}

	releaseCoordinatorNames = wf.ParamDef[[]string]{
		Name:      "Release Coordinator Names (optional)",
		ParamType: wf.SliceShort,
		Doc: `Release Coordinator Names is an optional list of release coordinator names to include in the sign-off message.

It shows up in the announcement mail.`,
	}
)

// newEchoWorkflow returns a runnable wf.Definition for
// development.
func newEchoWorkflow() *wf.Definition {
	wd := wf.New()
	wf.Output(wd, "greeting", wf.Task1(wd, "greeting", echo, wf.Param(wd, wf.ParamDef[string]{Name: "greeting"})))
	wf.Output(wd, "farewell", wf.Task1(wd, "farewell", echo, wf.Param(wd, wf.ParamDef[string]{Name: "farewell"})))
	return wd
}

func echo(ctx *wf.TaskContext, arg string) (string, error) {
	ctx.Printf("echo(%v, %q)", ctx, arg)
	return arg, nil
}

type AwaitConditionFunc func(ctx *wf.TaskContext) (done bool, err error)

// AwaitFunc is a wf.Task that polls the provided awaitCondition
// every period until it either returns true or returns an error.
func AwaitFunc(ctx *wf.TaskContext, period time.Duration, awaitCondition AwaitConditionFunc) error {
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			ok, err := awaitCondition(ctx)
			if ok || err != nil {
				return err
			}
		}
	}
}

func checkTaskApproved(ctx *wf.TaskContext, p *pgxpool.Pool) (bool, error) {
	q := db.New(p)
	t, err := q.Task(ctx, db.TaskParams{
		Name:       ctx.TaskName,
		WorkflowID: ctx.WorkflowID,
	})
	if !t.ReadyForApproval {
		_, err := q.UpdateTaskReadyForApproval(ctx, db.UpdateTaskReadyForApprovalParams{
			ReadyForApproval: true,
			Name:             ctx.TaskName,
			WorkflowID:       ctx.WorkflowID,
		})
		if err != nil {
			return false, err
		}
	}
	return t.ApprovedAt.Valid, err
}

// ApproveActionDep returns a function for defining approval Actions.
//
// ApproveActionDep takes a single *pgxpool.Pool argument, which is
// used to query the database to determine if a task has been marked
// approved.
//
// ApproveActionDep marks the task as requiring approval in the
// database once the task is started. This can be used to show an
// "approve" control in the UI.
//
//	waitAction := wf.ActionN(wd, "Wait for Approval", ApproveActionDep(db), wf.After(someDependency))
func ApproveActionDep(p *pgxpool.Pool) func(*wf.TaskContext) error {
	return func(ctx *wf.TaskContext) error {
		return AwaitFunc(ctx, 5*time.Second, func(ctx *wf.TaskContext) (done bool, err error) {
			return checkTaskApproved(ctx, p)
		})
	}
}

// RegisterReleaseWorkflows registers workflows for issuing Go releases.
func RegisterReleaseWorkflows(ctx context.Context, h *DefinitionHolder, build *BuildReleaseTasks, milestone *task.MilestoneTasks, version *task.VersionTasks, comm task.CommunicationTasks) error {
	// Register prod release workflows both with, and without comm tasks at the end.
	// TODO(go.dev/issue/53537): Simplify after more experience.
	if err := registerProdReleaseWorkflows(ctx, h, build, milestone, version, comm, true, ""); err != nil {
		return err
	}
	if err := registerProdReleaseWorkflows(ctx, h, build, milestone, version, comm, false, "[without comms] "); err != nil {
		return err
	}

	// Register dry-run release workflows.
	currentMajor, err := version.GetCurrentMajor(ctx)
	if err != nil {
		return err
	}
	wd := wf.New()
	if err := addBuildAndTestOnlyWorkflow(wd, version, build, currentMajor+1, task.KindBeta); err != nil {
		return err
	}
	h.RegisterDefinition(fmt.Sprintf("dry-run (test and build only): Go 1.%d next beta", currentMajor+1), wd)

	return nil
}

func registerProdReleaseWorkflows(ctx context.Context, h *DefinitionHolder, build *BuildReleaseTasks, milestone *task.MilestoneTasks, version *task.VersionTasks, comm task.CommunicationTasks, mergeCommTasks bool, definitionPrefix string) error {
	currentMajor, err := version.GetCurrentMajor(ctx)
	if err != nil {
		return err
	}
	releases := []struct {
		kind   task.ReleaseKind
		major  int
		suffix string
	}{
		{task.KindMajor, currentMajor + 1, "final"},
		{task.KindRC, currentMajor + 1, "next RC"},
		{task.KindBeta, currentMajor + 1, "next beta"},
		{task.KindCurrentMinor, currentMajor, "next minor"},
		{task.KindPrevMinor, currentMajor - 1, "next minor"},
	}
	for _, r := range releases {
		wd := wf.New()

		var securitySummary wf.Value[string]
		var securityFixes, names wf.Value[[]string]
		if mergeCommTasks {
			if r.kind == task.KindCurrentMinor || r.kind == task.KindPrevMinor {
				securitySummary = wf.Param(wd, securitySummaryParameter)
				securityFixes = wf.Param(wd, securityFixesParameter)
			}
			names = wf.Param(wd, releaseCoordinatorNames)
		}

		versionPublished, err := addSingleReleaseWorkflow(build, milestone, version, wd, r.major, r.kind)
		if err != nil {
			return err
		}

		if mergeCommTasks {
			addCommTasksForSingleRelease(wd, build, comm, r.kind, versionPublished, securitySummary, securityFixes, names)
		}

		h.RegisterDefinition(definitionPrefix+fmt.Sprintf("Go 1.%d %s", r.major, r.suffix), wd)
	}

	wd, err := createMinorReleaseWorkflow(build, milestone, version, comm, mergeCommTasks, currentMajor-1, currentMajor)
	if err != nil {
		return err
	}
	h.RegisterDefinition(definitionPrefix+fmt.Sprintf("Minor releases for Go 1.%d and 1.%d", currentMajor-1, currentMajor), wd)

	return nil
}

func addBuildAndTestOnlyWorkflow(wd *wf.Definition, version *task.VersionTasks, build *BuildReleaseTasks, major int, kind task.ReleaseKind) error {
	nextVersion := wf.Task1(wd, "Get next version", version.GetNextVersion, wf.Const(kind))
	branch := fmt.Sprintf("release-branch.go1.%d", major)
	if kind == task.KindBeta {
		branch = "master"
	}
	branchVal := wf.Const(branch)
	source := wf.Task3(wd, "Build source archive", build.buildSource, branchVal, wf.Const(""), nextVersion)
	artifacts, err := build.addBuildTasks(wd, major, nextVersion, source, true)
	if err != nil {
		return err
	}
	wf.Output(wd, "Artifacts", artifacts)
	return nil
}

func createMinorReleaseWorkflow(build *BuildReleaseTasks, milestone *task.MilestoneTasks, version *task.VersionTasks, comm task.CommunicationTasks, mergeCommTasks bool, prevMajor, currentMajor int) (*wf.Definition, error) {
	wd := wf.New()

	var securitySummary wf.Value[string]
	var securityFixes, names wf.Value[[]string]
	if mergeCommTasks {
		securitySummary = wf.Param(wd, securitySummaryParameter)
		securityFixes = wf.Param(wd, securityFixesParameter)
		names = wf.Param(wd, releaseCoordinatorNames)
	}

	v1Published, err := addSingleReleaseWorkflow(build, milestone, version, wd.Sub(fmt.Sprintf("Go 1.%d", currentMajor)), currentMajor, task.KindCurrentMinor)
	if err != nil {
		return nil, err
	}
	v2Published, err := addSingleReleaseWorkflow(build, milestone, version, wd.Sub(fmt.Sprintf("Go 1.%d", prevMajor)), prevMajor, task.KindPrevMinor)
	if err != nil {
		return nil, err
	}

	if mergeCommTasks {
		addCommTasksForDoubleMinorRelease(wd, build, comm, v1Published, v2Published, securitySummary, securityFixes, names)
	}

	return wd, nil
}

func addCommTasksForDoubleMinorRelease(
	wd *wf.Definition, build *BuildReleaseTasks, comm task.CommunicationTasks,
	v1Published, v2Published, securitySummary wf.Value[string], securityFixes, names wf.Value[[]string],
) {
	okayToAnnounceAndTweet := wf.Action0(wd, "Wait to Announce", build.ApproveAction, wf.After(v1Published, v2Published))

	// Announce that a new Go release has been published.
	sentMail := wf.Task4(wd, "mail-announcement", func(ctx *wf.TaskContext, v1, v2 string, sec, names []string) (task.SentMail, error) {
		return comm.AnnounceRelease(ctx, task.ReleaseAnnouncement{Version: v1, SecondaryVersion: v2, Security: sec, Names: names})
	}, v1Published, v2Published, securityFixes, names, wf.After(okayToAnnounceAndTweet))
	announcementURL := wf.Task1(wd, "await-announcement", comm.AwaitAnnounceMail, sentMail)
	tweetURL := wf.Task4(wd, "post-tweet", func(ctx *wf.TaskContext, v1, v2, sec, ann string) (string, error) {
		return comm.TweetRelease(ctx, task.ReleaseTweet{Version: v1, SecondaryVersion: v2, Security: sec, Announcement: ann})
	}, v1Published, v2Published, securitySummary, announcementURL, wf.After(okayToAnnounceAndTweet))

	wf.Output(wd, "Announcement URL", announcementURL)
	wf.Output(wd, "Tweet URL", tweetURL)
}
func addCommTasksForSingleRelease(
	wd *wf.Definition, build *BuildReleaseTasks, comm task.CommunicationTasks,
	kind task.ReleaseKind, versionPublished, securitySummary wf.Value[string], securityFixes, names wf.Value[[]string],
) {
	okayToAnnounceAndTweet := wf.Action0(wd, "Wait to Announce", build.ApproveAction, wf.After(versionPublished))

	// Announce that a new Go release has been published.
	var announcementURL, tweetURL wf.Value[string]
	if kind == task.KindCurrentMinor || kind == task.KindPrevMinor {
		sentMail := wf.Task3(wd, "mail-announcement", func(ctx *wf.TaskContext, v string, sec, names []string) (task.SentMail, error) {
			return comm.AnnounceRelease(ctx, task.ReleaseAnnouncement{Version: v, Security: sec, Names: names})
		}, versionPublished, securityFixes, names, wf.After(okayToAnnounceAndTweet))
		announcementURL = wf.Task1(wd, "await-announcement", comm.AwaitAnnounceMail, sentMail)
		tweetURL = wf.Task3(wd, "post-tweet", func(ctx *wf.TaskContext, v, sec, ann string) (string, error) {
			return comm.TweetRelease(ctx, task.ReleaseTweet{Version: v, Security: sec, Announcement: ann})
		}, versionPublished, securitySummary, announcementURL, wf.After(okayToAnnounceAndTweet))
	} else {
		sentMail := wf.Task2(wd, "mail-announcement", func(ctx *wf.TaskContext, v string, names []string) (task.SentMail, error) {
			return comm.AnnounceRelease(ctx, task.ReleaseAnnouncement{Version: v, Names: names})
		}, versionPublished, names, wf.After(okayToAnnounceAndTweet))
		announcementURL = wf.Task1(wd, "await-announcement", comm.AwaitAnnounceMail, sentMail)
		tweetURL = wf.Task2(wd, "post-tweet", func(ctx *wf.TaskContext, v, ann string) (string, error) {
			return comm.TweetRelease(ctx, task.ReleaseTweet{Version: v, Announcement: ann})
		}, versionPublished, announcementURL, wf.After(okayToAnnounceAndTweet))
	}

	wf.Output(wd, "Announcement URL", announcementURL)
	wf.Output(wd, "Tweet URL", tweetURL)
}

func addSingleReleaseWorkflow(
	build *BuildReleaseTasks, milestone *task.MilestoneTasks, version *task.VersionTasks,
	wd *wf.Definition, major int, kind task.ReleaseKind,
) (versionPublished wf.Value[string], _ error) {
	kindVal := wf.Const(kind)
	branch := fmt.Sprintf("release-branch.go1.%d", major)
	if kind == task.KindBeta {
		branch = "master"
	}
	branchVal := wf.Const(branch)
	startingHead := wf.Task1(wd, "Read starting branch head", version.ReadBranchHead, branchVal)

	// Select version, check milestones.
	nextVersion := wf.Task1(wd, "Get next version", version.GetNextVersion, kindVal)
	milestones := wf.Task2(wd, "Pick milestones", milestone.FetchMilestones, nextVersion, kindVal)
	checked := wf.Action3(wd, "Check blocking issues", milestone.CheckBlockers, milestones, nextVersion, kindVal)

	startSigner := wf.Task1(wd, "Start signing command", build.startSigningCommand, nextVersion)
	wf.Output(wd, "Signing command", startSigner)

	securityRef := wf.Param(wd, wf.ParamDef[string]{Name: "Ref from the private repository to build from (optional)"})
	source := wf.Task3(wd, "Build source archive", build.buildSource, startingHead, securityRef, nextVersion, wf.After(checked))

	// Build, test, and sign release.
	signedAndTestedArtifacts, err := build.addBuildTasks(wd, major, nextVersion, source, false)
	if err != nil {
		return nil, err
	}

	okayToTagAndPublish := wf.Action0(wd, "Wait for Release Coordinator Approval", build.ApproveAction, wf.After(signedAndTestedArtifacts))

	dlcl := wf.Task2(wd, "Mail DL CL", version.MailDLCL, wf.Slice(nextVersion), wf.Const(false), wf.After(okayToTagAndPublish))
	dlclCommit := wf.Task2(wd, "Wait for DL CL", version.AwaitCL, dlcl, wf.Const(""))
	wf.Output(wd, "Download CL submitted", dlclCommit)

	// Tag version and upload to CDN/website.
	// If we're releasing a beta from master, tagging is easy; we just tag the
	// commit we started from. Otherwise, we're going to submit a VERSION CL,
	// and we need to make sure that that CL is submitted on top of the same
	// state we built from. For security releases that state may not have
	// been public when we started, but it should be now.
	tagCommit := startingHead
	if branch != "master" {
		publishingHead := wf.Task1(wd, "Read current branch head", version.ReadBranchHead, branchVal, wf.After(okayToTagAndPublish))
		branchHeadChecked := wf.Action3(wd, "Check branch state matches source archive", build.checkSourceMatch, publishingHead, nextVersion, source)
		versionCL := wf.Task2(wd, "Mail version CL", version.CreateAutoSubmitVersionCL, branchVal, nextVersion, wf.After(branchHeadChecked))
		tagCommit = wf.Task2(wd, "Wait for version CL submission", version.AwaitCL, versionCL, publishingHead)
	}
	tagged := wf.Action2(wd, "Tag version", version.TagRelease, nextVersion, tagCommit, wf.After(okayToTagAndPublish))
	uploaded := wf.Action1(wd, "Upload artifacts to CDN", build.uploadArtifacts, signedAndTestedArtifacts, wf.After(tagged))
	pushed := wf.Action3(wd, "Push issues", milestone.PushIssues, milestones, nextVersion, kindVal, wf.After(tagged))
	versionPublished = wf.Task2(wd, "Publish to website", build.publishArtifacts, nextVersion, signedAndTestedArtifacts, wf.After(uploaded, pushed))
	wf.Output(wd, "Released version", versionPublished)
	return versionPublished, nil
}

// addBuildTasks registers tasks to build, test, and sign the release onto wd.
// It returns the output from the last task, a slice of signed and tested artifacts.
func (tasks *BuildReleaseTasks) addBuildTasks(wd *wf.Definition, major int, version wf.Value[string], source wf.Value[artifact], skipSigning bool) (wf.Value[[]artifact], error) {
	targets := releasetargets.TargetsForGo1Point(major)
	skipTests := wf.Param(wd, wf.ParamDef[[]string]{Name: "Targets to skip testing (or 'all') (optional)", ParamType: wf.SliceShort})
	// Artifact file paths.
	artifacts := []wf.Value[artifact]{source}
	var darwinTargets []*releasetargets.Target
	var testsPassed []wf.Dependency
	for _, target := range targets {
		targetVal := wf.Const(target)
		wd := wd.Sub(target.Name)

		// Build release artifacts for the platform.
		bin := wf.Task2(wd, "Build binary archive", tasks.buildBinary, targetVal, source)
		switch target.GOOS {
		case "windows":
			zip := wf.Task2(wd, "Convert to .zip", tasks.convertToZip, targetVal, bin)
			msi := wf.Task2(wd, "Build MSI", tasks.buildMSI, targetVal, bin)
			artifacts = append(artifacts, msi, zip)
		case "darwin":
			artifacts = append(artifacts, bin)
			darwinTargets = append(darwinTargets, target)
		default:
			artifacts = append(artifacts, bin)
		}

		if target.BuildOnly {
			continue
		}
		short := wf.Action4(wd, "Run short tests", tasks.runTests, targetVal, wf.Const(dashboard.Builders[target.Builder]), skipTests, bin)
		testsPassed = append(testsPassed, short)
		if target.LongTestBuilder != "" {
			long := wf.Action4(wd, "Run long tests", tasks.runTests, targetVal, wf.Const(dashboard.Builders[target.Builder]), skipTests, bin)
			testsPassed = append(testsPassed, long)
		}
	}
	var advisoryResults []wf.Value[tryBotResult]
	for _, bc := range advisoryTryBots(major) {
		result := wf.Task3(wd, "Run advisory TryBot "+bc.Name, tasks.runAdvisoryTryBot, wf.Const(bc), skipTests, source)
		advisoryResults = append(advisoryResults, result)
	}
	tryBotsApproved := wf.Action1(wd, "Approve any TryBot failures", tasks.checkAdvisoryTrybots, wf.Slice(advisoryResults...))
	if skipSigning {
		builtAndTested := wf.Task1(wd, "Wait for artifacts and tests", func(ctx *wf.TaskContext, artifacts []artifact) ([]artifact, error) {
			return artifacts, nil
		}, wf.Slice(artifacts...), wf.After(tryBotsApproved), wf.After(testsPassed...))
		return builtAndTested, nil
	}
	stagedArtifacts := wf.Task2(wd, "Stage artifacts for signing", tasks.copyToStaging, version, wf.Slice(artifacts...))
	signedArtifacts := wf.Task3(wd, "Wait for signed artifacts", tasks.awaitSigned, version, wf.Const(darwinTargets), stagedArtifacts)
	signedAndTested := wf.Task1(wd, "Wait for signing and tests", func(ctx *wf.TaskContext, artifacts []artifact) ([]artifact, error) {
		return artifacts, nil
	}, signedArtifacts, wf.After(tryBotsApproved), wf.After(testsPassed...))
	return signedAndTested, nil
}

func advisoryTryBots(major int) []*dashboard.BuildConfig {
	usedBuilders := map[string]bool{}
	for _, t := range releasetargets.TargetsForGo1Point(major) {
		usedBuilders[t.Builder] = true
		usedBuilders[t.LongTestBuilder] = true
	}

	var extras []*dashboard.BuildConfig
	for name, bc := range dashboard.Builders {
		if usedBuilders[name] {
			continue
		}
		if !bc.BuildsRepoPostSubmit("go", fmt.Sprintf("release-branch.go1.%d", major), "") {
			continue
		}
		if !bc.IsVM() && !bc.IsContainer() {
			continue
		}
		extras = append(extras, bc)
	}
	return extras
}

// BuildReleaseTasks serves as an adapter to the various build tasks in the task package.
type BuildReleaseTasks struct {
	GerritHTTPClient       *http.Client
	GerritURL              string
	PrivateGerritURL       string
	GCSClient              *storage.Client
	ScratchURL, ServingURL string
	DownloadURL            string
	PublishFile            func(*WebsiteFile) error
	CreateBuildlet         func(string) (buildlet.Client, error)
	ApproveAction          func(*wf.TaskContext) error
}

func (b *BuildReleaseTasks) buildSource(ctx *wf.TaskContext, revision, securityRevision, version string) (artifact, error) {
	return b.runBuildStep(ctx, nil, nil, artifact{}, "src.tar.gz", func(_ *task.BuildletStep, _ io.Reader, w io.Writer) error {
		if securityRevision != "" {
			return task.WriteSourceArchive(ctx, b.GerritHTTPClient, b.PrivateGerritURL, securityRevision, version, w)
		}
		return task.WriteSourceArchive(ctx, b.GerritHTTPClient, b.GerritURL, revision, version, w)
	})
}

func (b *BuildReleaseTasks) checkSourceMatch(ctx *wf.TaskContext, head, version string, source artifact) error {
	_, err := b.runBuildStep(ctx, nil, nil, source, "", func(_ *task.BuildletStep, r io.Reader, _ io.Writer) error {
		branchArchive := &bytes.Buffer{}
		if err := task.WriteSourceArchive(ctx, b.GerritHTTPClient, b.GerritURL, head, version, branchArchive); err != nil {
			return err
		}
		branchHashes, err := tarballHashes(branchArchive)
		if err != nil {
			return fmt.Errorf("hashing branch tarball: %v", err)
		}
		archiveHashes, err := tarballHashes(r)
		if err != nil {
			return fmt.Errorf("hashing archive tarball: %v", err)
		}
		if diff := cmp.Diff(branchHashes, archiveHashes); diff != "" {
			return fmt.Errorf("branch state doesn't match source archive (-branch, +archive):\n%v", diff)
		}
		return nil
	})
	return err
}

func tarballHashes(r io.Reader) (map[string]string, error) {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	defer gzr.Close()
	tr := tar.NewReader(gzr)
	hashes := map[string]string{}
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("reading tar header: %v", err)
		}
		h := sha256.New()
		if _, err := io.CopyN(h, tr, header.Size); err != nil {
			return nil, fmt.Errorf("reading file %q: %v", header.Name, err)
		}
		hashes[header.Name] = fmt.Sprintf("%X", h.Sum(nil))
	}
	return hashes, nil
}

func (b *BuildReleaseTasks) buildBinary(ctx *wf.TaskContext, target *releasetargets.Target, source artifact) (artifact, error) {
	build := dashboard.Builders[target.Builder]
	return b.runBuildStep(ctx, target, build, source, "tar.gz", func(bs *task.BuildletStep, r io.Reader, w io.Writer) error {
		return bs.BuildBinary(ctx, r, w)
	})
}

func (b *BuildReleaseTasks) buildMSI(ctx *wf.TaskContext, target *releasetargets.Target, binary artifact) (artifact, error) {
	build := dashboard.Builders[target.Builder]
	return b.runBuildStep(ctx, target, build, binary, "msi", func(bs *task.BuildletStep, r io.Reader, w io.Writer) error {
		return bs.BuildMSI(ctx, r, w)
	})
}

func (b *BuildReleaseTasks) convertToZip(ctx *wf.TaskContext, target *releasetargets.Target, binary artifact) (artifact, error) {
	return b.runBuildStep(ctx, target, nil, binary, "zip", func(_ *task.BuildletStep, r io.Reader, w io.Writer) error {
		return task.ConvertTGZToZIP(r, w)
	})
}

func (b *BuildReleaseTasks) runTests(ctx *wf.TaskContext, target *releasetargets.Target, build *dashboard.BuildConfig, skipTests []string, binary artifact) error {
	for _, skip := range skipTests {
		if skip == "all" || target.Name == skip {
			ctx.Printf("Skipping test")
			return nil
		}
	}
	_, err := b.runBuildStep(ctx, target, build, binary, "", func(bs *task.BuildletStep, r io.Reader, _ io.Writer) error {
		return bs.TestTarget(ctx, r)
	})
	return err
}

type tryBotResult struct {
	Name   string
	Passed bool
}

func (b *BuildReleaseTasks) runAdvisoryTryBot(ctx *wf.TaskContext, bc *dashboard.BuildConfig, skipTests []string, source artifact) (tryBotResult, error) {
	for _, skip := range skipTests {
		if skip == "all" || bc.Name == skip {
			ctx.Printf("Skipping test")
			return tryBotResult{bc.Name, true}, nil
		}
	}

	passed := false
	_, err := b.runBuildStep(ctx, nil, bc, source, "", func(bs *task.BuildletStep, r io.Reader, w io.Writer) error {
		var err error
		passed, err = bs.RunTryBot(ctx, r)
		return err
	})
	return tryBotResult{bc.Name, passed}, err
}

func (b *BuildReleaseTasks) checkAdvisoryTrybots(ctx *wf.TaskContext, results []tryBotResult) error {
	var fails []string
	for _, r := range results {
		if !r.Passed {
			fails = append(fails, r.Name)
		}
	}
	if len(fails) == 0 {
		return nil
	}
	ctx.Printf("Some advisory TryBots failed. Check their logs and approve this task if it's okay:\n%v", strings.Join(fails, "\n"))
	return b.ApproveAction(ctx)
}

// runBuildStep is a convenience function that manages resources a build step might need.
// If target and build config are specified, a BuildletStep will be passed to f.
// If inputName is specified, it will be opened and passed as a Reader to f.
// If outputSuffix is specified, a unique filename will be generated based off
// it (and the target name, if any), the file will be opened and passed as a
// Writer to f, and an artifact representing it will be returned as the result.
func (b *BuildReleaseTasks) runBuildStep(
	ctx *wf.TaskContext,
	target *releasetargets.Target,
	build *dashboard.BuildConfig,
	input artifact,
	outputSuffix string,
	f func(*task.BuildletStep, io.Reader, io.Writer) error,
) (artifact, error) {
	var step *task.BuildletStep
	if build != nil {
		ctx.Printf("Creating buildlet %v.", build.Name)
		client, err := b.CreateBuildlet(build.Name)
		if err != nil {
			return artifact{}, err
		}
		defer client.Close()
		step = &task.BuildletStep{
			Target:      target,
			Buildlet:    client,
			BuildConfig: build,
			Watch:       true,
		}
		ctx.Printf("Buildlet ready.")
	}

	scratchFS, err := gcsfs.FromURL(ctx, b.GCSClient, b.ScratchURL)
	if err != nil {
		return artifact{}, err
	}
	var in io.ReadCloser
	if input.ScratchPath != "" {
		in, err = scratchFS.Open(input.ScratchPath)
		if err != nil {
			return artifact{}, err
		}
		defer in.Close()
	}
	var out io.WriteCloser
	var scratchPath string
	hash := sha256.New()
	size := &sizeWriter{}
	var multiOut io.Writer
	if outputSuffix != "" {
		scratchName := outputSuffix
		if target != nil {
			scratchName = target.Name + "." + outputSuffix
		}
		scratchPath = fmt.Sprintf("%v/%v-%v", ctx.WorkflowID.String(), scratchName, rand.Int63())
		out, err = gcsfs.Create(scratchFS, scratchPath)
		if err != nil {
			return artifact{}, err
		}
		defer out.Close()
		multiOut = io.MultiWriter(out, hash, size)
	}
	// Hide in's Close method from the task, which may assert it to Closer.
	nopIn := io.NopCloser(in)
	if err := f(step, nopIn, multiOut); err != nil {
		return artifact{}, err
	}
	if step != nil {
		if err := step.Buildlet.Close(); err != nil {
			return artifact{}, err
		}
	}
	if in != nil {
		if err := in.Close(); err != nil {
			return artifact{}, err
		}
	}
	if out != nil {
		if err := out.Close(); err != nil {
			return artifact{}, err
		}
	}
	return artifact{
		Target:      target,
		ScratchPath: scratchPath,
		Suffix:      outputSuffix,
		SHA256:      fmt.Sprintf("%x", string(hash.Sum([]byte(nil)))),
		Size:        size.size,
	}, nil
}

// An artifact represents a file as it moves through the release process. Most
// files will appear on go.dev/dl eventually.
type artifact struct {
	// The target platform of this artifact, or nil for source.
	Target *releasetargets.Target
	// The scratch path of this artifact within the scratch directory.
	// <workflow-id>/<filename>-<random-number>
	ScratchPath string
	// The path within the scratch directory the artifact was staged to for the
	// signing process.
	// <workflow-id>/signing/<go version>/<filename>
	StagingPath string
	// The path within the scratch directory the artifact can be found at
	// after the signing process. For files not modified by the signing
	// process, the staging path, or for those that are
	// <workflow-id>/signing/<go version>/signed/<filename>
	SignedPath string
	// The contents of the GPG signature for this artifact (.asc file).
	GPGSignature string
	// The filename suffix of the artifact, e.g. "tar.gz" or "src.tar.gz",
	// combined with the version and Target name to produce Filename.
	Suffix string
	// The final Filename of this artifact as it will be downloaded.
	Filename string
	SHA256   string
	Size     int
}

type sizeWriter struct {
	size int
}

func (w *sizeWriter) Write(p []byte) (n int, err error) {
	w.size += len(p)
	return len(p), nil
}

func (tasks *BuildReleaseTasks) startSigningCommand(ctx *wf.TaskContext, version string) (string, error) {
	args := fmt.Sprintf("--relui_staging=%q", tasks.ScratchURL+"/"+signingStagingDir(ctx, version))
	ctx.Printf("run signer with " + args)
	return args, nil
}

func (tasks *BuildReleaseTasks) copyToStaging(ctx *wf.TaskContext, version string, artifacts []artifact) ([]artifact, error) {
	scratchFS, err := gcsfs.FromURL(ctx, tasks.GCSClient, tasks.ScratchURL)
	if err != nil {
		return nil, err
	}
	var stagedArtifacts []artifact
	for _, a := range artifacts {
		staged := a
		if a.Target != nil {
			staged.Filename = version + "." + a.Target.Name + "." + a.Suffix
		} else {
			staged.Filename = version + "." + a.Suffix
		}
		staged.StagingPath = path.Join(signingStagingDir(ctx, version), staged.Filename)
		stagedArtifacts = append(stagedArtifacts, staged)

		in, err := scratchFS.Open(a.ScratchPath)
		if err != nil {
			return nil, err
		}
		out, err := gcsfs.Create(scratchFS, staged.StagingPath)
		if err != nil {
			return nil, err
		}
		if _, err := io.Copy(out, in); err != nil {
			return nil, err
		}
		if err := in.Close(); err != nil {
			return nil, err
		}
		if err := out.Close(); err != nil {
			return nil, err
		}
	}
	out, err := gcsfs.Create(scratchFS, path.Join(signingStagingDir(ctx, version), "ready"))
	if err != nil {
		return nil, err
	}
	if _, err := out.Write([]byte("ready")); err != nil {
		return nil, err
	}
	if err := out.Close(); err != nil {
		return nil, err
	}
	return stagedArtifacts, nil
}

func signingStagingDir(ctx *wf.TaskContext, version string) string {
	return path.Join(ctx.WorkflowID.String(), "signing", version)
}

var signingPollDuration = 30 * time.Second

// awaitSigned waits for all of artifacts to be signed, plus the pkgs for
// darwinTargets.
func (tasks *BuildReleaseTasks) awaitSigned(ctx *wf.TaskContext, version string, darwinTargets []*releasetargets.Target, artifacts []artifact) ([]artifact, error) {
	// .pkg artifacts are created by the signing process. Create placeholders,
	// to be filled out once the files exist.
	for _, t := range darwinTargets {
		artifacts = append(artifacts, artifact{
			Target:   t,
			Suffix:   "pkg",
			Filename: version + "." + t.Name + ".pkg",
			Size:     -1,
		})
	}

	scratchFS, err := gcsfs.FromURL(ctx, tasks.GCSClient, tasks.ScratchURL)
	if err != nil {
		return nil, err
	}

	todo := map[artifact]bool{}
	for _, a := range artifacts {
		todo[a] = true
	}
	var signedArtifacts []artifact
	for {
		for a := range todo {
			signed, ok, err := readSignedArtifact(ctx, scratchFS, version, a)
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}

			signedArtifacts = append(signedArtifacts, signed)
			delete(todo, a)
		}

		if len(todo) == 0 {
			return signedArtifacts, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(signingPollDuration):
			ctx.Printf("Still waiting for %v artifacts to be signed", len(todo))
		}
	}
}

func readSignedArtifact(ctx *wf.TaskContext, scratchFS fs.FS, version string, a artifact) (_ artifact, ok bool, _ error) {
	// Our signing process has somewhat uneven behavior. In general, for things
	// that contain their own signature, such as MSIs and .pkgs, we don't
	// produce a GPG signature, just the new file. On macOS, tars can be signed
	// too, but we GPG sign them anyway.
	modifiedBySigning := false
	hasGPG := false
	suffix := func(suffix string) bool { return a.Suffix == suffix }
	switch {
	case suffix("src.tar.gz"):
		hasGPG = true
	case a.Target.GOOS == "darwin" && suffix("tar.gz"):
		modifiedBySigning = true
		hasGPG = true
	case a.Target.GOOS == "darwin" && suffix("pkg"):
		modifiedBySigning = true
	case suffix("tar.gz"):
		hasGPG = true
	case suffix("msi"):
		modifiedBySigning = true
	case suffix("zip"):
		// For reasons unclear, we don't sign zip files.
	default:
		return artifact{}, false, fmt.Errorf("unhandled file type %q", a.Suffix)
	}

	signed := artifact{
		Target:   a.Target,
		Filename: a.Filename,
		Suffix:   a.Suffix,
	}
	stagingDir := signingStagingDir(ctx, version)
	if modifiedBySigning {
		signed.SignedPath = stagingDir + "/signed/" + a.Filename
	} else {
		signed.SignedPath = stagingDir + "/" + a.Filename
	}

	fi, err := fs.Stat(scratchFS, signed.SignedPath)
	if err != nil {
		return artifact{}, false, nil
	}
	if modifiedBySigning {
		hash, err := fs.ReadFile(scratchFS, stagingDir+"/signed/"+a.Filename+".sha256")
		if err != nil {
			return artifact{}, false, nil
		}
		signed.Size = int(fi.Size())
		signed.SHA256 = string(hash)
	} else {
		signed.SHA256 = a.SHA256
		signed.Size = a.Size
	}
	if hasGPG {
		sig, err := fs.ReadFile(scratchFS, stagingDir+"/signed/"+a.Filename+".asc")
		if err != nil {
			return artifact{}, false, nil
		}
		signed.GPGSignature = string(sig)
	}
	return signed, true, nil
}

var uploadPollDuration = 30 * time.Second

func (tasks *BuildReleaseTasks) uploadArtifacts(ctx *wf.TaskContext, artifacts []artifact) error {
	scratchFS, err := gcsfs.FromURL(ctx, tasks.GCSClient, tasks.ScratchURL)
	if err != nil {
		return err
	}
	servingFS, err := gcsfs.FromURL(ctx, tasks.GCSClient, tasks.ServingURL)
	if err != nil {
		return err
	}

	todo := map[artifact]bool{}
	for _, a := range artifacts {
		if err := uploadArtifact(scratchFS, servingFS, a); err != nil {
			return err
		}
		todo[a] = true
	}

	for {
		for _, a := range artifacts {
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			resp, err := ctxhttp.Head(ctx, http.DefaultClient, tasks.DownloadURL+"/"+a.Filename)
			if err != nil && err != context.DeadlineExceeded {
				return err
			}
			resp.Body.Close()
			cancel()
			if resp.StatusCode == http.StatusOK {
				delete(todo, a)
			}
		}

		if len(todo) == 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(uploadPollDuration):
			ctx.Printf("Still waiting for %v artifacts to be published", len(todo))
		}
	}
}

func uploadArtifact(scratchFS, servingFS fs.FS, a artifact) error {
	in, err := scratchFS.Open(a.SignedPath)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := gcsfs.Create(servingFS, a.Filename)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}

	sha256, err := gcsfs.Create(servingFS, a.Filename+".sha256")
	if err != nil {
		return err
	}
	defer sha256.Close()
	if _, err := sha256.Write([]byte(a.SHA256)); err != nil {
		return err
	}
	if err := sha256.Close(); err != nil {
		return err
	}

	if a.GPGSignature != "" {
		asc, err := gcsfs.Create(servingFS, a.Filename+".asc")
		if err != nil {
			return err
		}
		defer asc.Close()
		if _, err := asc.Write([]byte(a.GPGSignature)); err != nil {
			return err
		}
		if err := asc.Close(); err != nil {
			return err
		}
	}
	return nil
}

// publishArtifacts publishes artifacts for version (typically so they appear at https://go.dev/dl/).
// It returns version, the Go version that has been successfully published.
//
// The version string uses the same format as Go tags. For example, "go1.19rc1".
func (tasks *BuildReleaseTasks) publishArtifacts(ctx *wf.TaskContext, version string, artifacts []artifact) (publishedVersion string, _ error) {
	for _, a := range artifacts {
		f := &WebsiteFile{
			Filename:       a.Filename,
			Version:        version,
			ChecksumSHA256: a.SHA256,
			Size:           int64(a.Size),
		}
		if a.Target != nil {
			f.OS = a.Target.GOOS
			f.Arch = a.Target.GOARCH
			if a.Target.GOARCH == "arm" {
				f.Arch = "armv6l"
			}
		}
		switch a.Suffix {
		case "src.tar.gz":
			f.Kind = "source"
		case "tar.gz", "zip":
			f.Kind = "archive"
		case "msi", "pkg":
			f.Kind = "installer"
		}
		if err := tasks.PublishFile(f); err != nil {
			return "", err
		}
	}
	if log := ctx.Logger; log != nil {
		log.Printf("Published %v artifacts for %v", len(artifacts), version)
	}
	return version, nil
}

// WebsiteFile represents a file on the go.dev downloads page.
// It should be kept in sync with the download code in x/website/internal/dl.
type WebsiteFile struct {
	Filename       string `json:"filename"`
	OS             string `json:"os"`
	Arch           string `json:"arch"`
	Version        string `json:"version"`
	ChecksumSHA256 string `json:"sha256"`
	Size           int64  `json:"size"`
	Kind           string `json:"kind"` // "archive", "installer", "source"
}
