// Code generated by protoc-gen-go4grpc; DO NOT EDIT
// source: maintner.proto

/*
Package maintpb is a generated protocol buffer package.

It is generated from these files:
	maintner.proto

It has these top-level messages:
	Mutation
	GithubMutation
	GithubIssueMutation
	BoolChange
	GithubLabel
	GithubMilestone
	GithubIssueEvent
	GithubCommit
	GithubIssueSyncStatus
	GithubIssueCommentMutation
	GithubUser
	GitMutation
	GitRepo
	GitCommit
	GitDiffTree
	GitDiffTreeFile
	GerritMutation
	GitRef
*/
package maintpb

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import google_protobuf "github.com/golang/protobuf/ptypes/timestamp"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type Mutation struct {
	GithubIssue *GithubIssueMutation `protobuf:"bytes,1,opt,name=github_issue,json=githubIssue" json:"github_issue,omitempty"`
	Github      *GithubMutation      `protobuf:"bytes,3,opt,name=github" json:"github,omitempty"`
	Git         *GitMutation         `protobuf:"bytes,2,opt,name=git" json:"git,omitempty"`
	Gerrit      *GerritMutation      `protobuf:"bytes,4,opt,name=gerrit" json:"gerrit,omitempty"`
}

func (m *Mutation) Reset()                    { *m = Mutation{} }
func (m *Mutation) String() string            { return proto.CompactTextString(m) }
func (*Mutation) ProtoMessage()               {}
func (*Mutation) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *Mutation) GetGithubIssue() *GithubIssueMutation {
	if m != nil {
		return m.GithubIssue
	}
	return nil
}

func (m *Mutation) GetGithub() *GithubMutation {
	if m != nil {
		return m.Github
	}
	return nil
}

func (m *Mutation) GetGit() *GitMutation {
	if m != nil {
		return m.Git
	}
	return nil
}

func (m *Mutation) GetGerrit() *GerritMutation {
	if m != nil {
		return m.Gerrit
	}
	return nil
}

type GithubMutation struct {
	Owner string `protobuf:"bytes,1,opt,name=owner" json:"owner,omitempty"`
	Repo  string `protobuf:"bytes,2,opt,name=repo" json:"repo,omitempty"`
	// Updated labels. (All must have id set at least)
	Labels []*GithubLabel `protobuf:"bytes,3,rep,name=labels" json:"labels,omitempty"`
	// Updated milestones. (All must have id set at least)
	Milestones []*GithubMilestone `protobuf:"bytes,4,rep,name=milestones" json:"milestones,omitempty"`
}

func (m *GithubMutation) Reset()                    { *m = GithubMutation{} }
func (m *GithubMutation) String() string            { return proto.CompactTextString(m) }
func (*GithubMutation) ProtoMessage()               {}
func (*GithubMutation) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *GithubMutation) GetOwner() string {
	if m != nil {
		return m.Owner
	}
	return ""
}

func (m *GithubMutation) GetRepo() string {
	if m != nil {
		return m.Repo
	}
	return ""
}

func (m *GithubMutation) GetLabels() []*GithubLabel {
	if m != nil {
		return m.Labels
	}
	return nil
}

func (m *GithubMutation) GetMilestones() []*GithubMilestone {
	if m != nil {
		return m.Milestones
	}
	return nil
}

type GithubIssueMutation struct {
	Owner  string `protobuf:"bytes,1,opt,name=owner" json:"owner,omitempty"`
	Repo   string `protobuf:"bytes,2,opt,name=repo" json:"repo,omitempty"`
	Number int32  `protobuf:"varint,3,opt,name=number" json:"number,omitempty"`
	// not_exist is set true if the issue has been found to not exist.
	// If true, the owner/repo/number fields above must still be set.
	// If a future issue mutation for the same number arrives without
	// not_exist set, then the issue comes back to life.
	NotExist         bool                       `protobuf:"varint,13,opt,name=not_exist,json=notExist" json:"not_exist,omitempty"`
	Id               int64                      `protobuf:"varint,12,opt,name=id" json:"id,omitempty"`
	User             *GithubUser                `protobuf:"bytes,4,opt,name=user" json:"user,omitempty"`
	Assignees        []*GithubUser              `protobuf:"bytes,10,rep,name=assignees" json:"assignees,omitempty"`
	DeletedAssignees []int64                    `protobuf:"varint,11,rep,packed,name=deleted_assignees,json=deletedAssignees" json:"deleted_assignees,omitempty"`
	Created          *google_protobuf.Timestamp `protobuf:"bytes,5,opt,name=created" json:"created,omitempty"`
	Updated          *google_protobuf.Timestamp `protobuf:"bytes,6,opt,name=updated" json:"updated,omitempty"`
	Body             string                     `protobuf:"bytes,7,opt,name=body" json:"body,omitempty"`
	Title            string                     `protobuf:"bytes,9,opt,name=title" json:"title,omitempty"`
	NoMilestone      bool                       `protobuf:"varint,15,opt,name=no_milestone,json=noMilestone" json:"no_milestone,omitempty"`
	// When setting a milestone, only the milestone_id must be set.
	// TODO: allow num or title to be used if Github only returns those? So far unneeded.
	// The num and title, if non-zero, are treated as if they were a GithubMutation.Milestone update.
	MilestoneId    int64                         `protobuf:"varint,16,opt,name=milestone_id,json=milestoneId" json:"milestone_id,omitempty"`
	MilestoneNum   int64                         `protobuf:"varint,17,opt,name=milestone_num,json=milestoneNum" json:"milestone_num,omitempty"`
	MilestoneTitle string                        `protobuf:"bytes,18,opt,name=milestone_title,json=milestoneTitle" json:"milestone_title,omitempty"`
	Closed         *BoolChange                   `protobuf:"bytes,19,opt,name=closed" json:"closed,omitempty"`
	Locked         *BoolChange                   `protobuf:"bytes,25,opt,name=locked" json:"locked,omitempty"`
	PullRequest    bool                          `protobuf:"varint,28,opt,name=pull_request,json=pullRequest" json:"pull_request,omitempty"`
	ClosedAt       *google_protobuf.Timestamp    `protobuf:"bytes,21,opt,name=closed_at,json=closedAt" json:"closed_at,omitempty"`
	ClosedBy       *GithubUser                   `protobuf:"bytes,22,opt,name=closed_by,json=closedBy" json:"closed_by,omitempty"`
	RemoveLabel    []int64                       `protobuf:"varint,23,rep,packed,name=remove_label,json=removeLabel" json:"remove_label,omitempty"`
	AddLabel       []*GithubLabel                `protobuf:"bytes,24,rep,name=add_label,json=addLabel" json:"add_label,omitempty"`
	Comment        []*GithubIssueCommentMutation `protobuf:"bytes,8,rep,name=comment" json:"comment,omitempty"`
	CommentStatus  *GithubIssueSyncStatus        `protobuf:"bytes,14,opt,name=comment_status,json=commentStatus" json:"comment_status,omitempty"`
	Event          []*GithubIssueEvent           `protobuf:"bytes,26,rep,name=event" json:"event,omitempty"`
	EventStatus    *GithubIssueSyncStatus        `protobuf:"bytes,27,opt,name=event_status,json=eventStatus" json:"event_status,omitempty"`
}

func (m *GithubIssueMutation) Reset()                    { *m = GithubIssueMutation{} }
func (m *GithubIssueMutation) String() string            { return proto.CompactTextString(m) }
func (*GithubIssueMutation) ProtoMessage()               {}
func (*GithubIssueMutation) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

func (m *GithubIssueMutation) GetOwner() string {
	if m != nil {
		return m.Owner
	}
	return ""
}

func (m *GithubIssueMutation) GetRepo() string {
	if m != nil {
		return m.Repo
	}
	return ""
}

func (m *GithubIssueMutation) GetNumber() int32 {
	if m != nil {
		return m.Number
	}
	return 0
}

func (m *GithubIssueMutation) GetNotExist() bool {
	if m != nil {
		return m.NotExist
	}
	return false
}

func (m *GithubIssueMutation) GetId() int64 {
	if m != nil {
		return m.Id
	}
	return 0
}

func (m *GithubIssueMutation) GetUser() *GithubUser {
	if m != nil {
		return m.User
	}
	return nil
}

func (m *GithubIssueMutation) GetAssignees() []*GithubUser {
	if m != nil {
		return m.Assignees
	}
	return nil
}

func (m *GithubIssueMutation) GetDeletedAssignees() []int64 {
	if m != nil {
		return m.DeletedAssignees
	}
	return nil
}

func (m *GithubIssueMutation) GetCreated() *google_protobuf.Timestamp {
	if m != nil {
		return m.Created
	}
	return nil
}

func (m *GithubIssueMutation) GetUpdated() *google_protobuf.Timestamp {
	if m != nil {
		return m.Updated
	}
	return nil
}

func (m *GithubIssueMutation) GetBody() string {
	if m != nil {
		return m.Body
	}
	return ""
}

func (m *GithubIssueMutation) GetTitle() string {
	if m != nil {
		return m.Title
	}
	return ""
}

func (m *GithubIssueMutation) GetNoMilestone() bool {
	if m != nil {
		return m.NoMilestone
	}
	return false
}

func (m *GithubIssueMutation) GetMilestoneId() int64 {
	if m != nil {
		return m.MilestoneId
	}
	return 0
}

func (m *GithubIssueMutation) GetMilestoneNum() int64 {
	if m != nil {
		return m.MilestoneNum
	}
	return 0
}

func (m *GithubIssueMutation) GetMilestoneTitle() string {
	if m != nil {
		return m.MilestoneTitle
	}
	return ""
}

func (m *GithubIssueMutation) GetClosed() *BoolChange {
	if m != nil {
		return m.Closed
	}
	return nil
}

func (m *GithubIssueMutation) GetLocked() *BoolChange {
	if m != nil {
		return m.Locked
	}
	return nil
}

func (m *GithubIssueMutation) GetPullRequest() bool {
	if m != nil {
		return m.PullRequest
	}
	return false
}

func (m *GithubIssueMutation) GetClosedAt() *google_protobuf.Timestamp {
	if m != nil {
		return m.ClosedAt
	}
	return nil
}

func (m *GithubIssueMutation) GetClosedBy() *GithubUser {
	if m != nil {
		return m.ClosedBy
	}
	return nil
}

func (m *GithubIssueMutation) GetRemoveLabel() []int64 {
	if m != nil {
		return m.RemoveLabel
	}
	return nil
}

func (m *GithubIssueMutation) GetAddLabel() []*GithubLabel {
	if m != nil {
		return m.AddLabel
	}
	return nil
}

func (m *GithubIssueMutation) GetComment() []*GithubIssueCommentMutation {
	if m != nil {
		return m.Comment
	}
	return nil
}

func (m *GithubIssueMutation) GetCommentStatus() *GithubIssueSyncStatus {
	if m != nil {
		return m.CommentStatus
	}
	return nil
}

func (m *GithubIssueMutation) GetEvent() []*GithubIssueEvent {
	if m != nil {
		return m.Event
	}
	return nil
}

func (m *GithubIssueMutation) GetEventStatus() *GithubIssueSyncStatus {
	if m != nil {
		return m.EventStatus
	}
	return nil
}

// BoolChange represents a change to a boolean value.
type BoolChange struct {
	Val bool `protobuf:"varint,1,opt,name=val" json:"val,omitempty"`
}

func (m *BoolChange) Reset()                    { *m = BoolChange{} }
func (m *BoolChange) String() string            { return proto.CompactTextString(m) }
func (*BoolChange) ProtoMessage()               {}
func (*BoolChange) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

func (m *BoolChange) GetVal() bool {
	if m != nil {
		return m.Val
	}
	return false
}

type GithubLabel struct {
	Id   int64  `protobuf:"varint,1,opt,name=id" json:"id,omitempty"`
	Name string `protobuf:"bytes,2,opt,name=name" json:"name,omitempty"`
}

func (m *GithubLabel) Reset()                    { *m = GithubLabel{} }
func (m *GithubLabel) String() string            { return proto.CompactTextString(m) }
func (*GithubLabel) ProtoMessage()               {}
func (*GithubLabel) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{4} }

func (m *GithubLabel) GetId() int64 {
	if m != nil {
		return m.Id
	}
	return 0
}

func (m *GithubLabel) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

type GithubMilestone struct {
	Id int64 `protobuf:"varint,1,opt,name=id" json:"id,omitempty"`
	// Following only need to be non-zero on changes:
	Title  string      `protobuf:"bytes,2,opt,name=title" json:"title,omitempty"`
	Closed *BoolChange `protobuf:"bytes,3,opt,name=closed" json:"closed,omitempty"`
	Number int64       `protobuf:"varint,4,opt,name=number" json:"number,omitempty"`
}

func (m *GithubMilestone) Reset()                    { *m = GithubMilestone{} }
func (m *GithubMilestone) String() string            { return proto.CompactTextString(m) }
func (*GithubMilestone) ProtoMessage()               {}
func (*GithubMilestone) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{5} }

func (m *GithubMilestone) GetId() int64 {
	if m != nil {
		return m.Id
	}
	return 0
}

func (m *GithubMilestone) GetTitle() string {
	if m != nil {
		return m.Title
	}
	return ""
}

func (m *GithubMilestone) GetClosed() *BoolChange {
	if m != nil {
		return m.Closed
	}
	return nil
}

func (m *GithubMilestone) GetNumber() int64 {
	if m != nil {
		return m.Number
	}
	return 0
}

// See https://developer.github.com/v3/activity/events/types/#issuesevent
// for some info.
type GithubIssueEvent struct {
	// Required:
	Id int64 `protobuf:"varint,1,opt,name=id" json:"id,omitempty"`
	// event_type can be one of "assigned", "unassigned", "labeled",
	// "unlabeled", "opened", "edited", "milestoned", "demilestoned",
	// "closed", "reopened", "referenced", "renamed" or anything else
	// that Github adds in the future.
	EventType string                     `protobuf:"bytes,2,opt,name=event_type,json=eventType" json:"event_type,omitempty"`
	ActorId   int64                      `protobuf:"varint,3,opt,name=actor_id,json=actorId" json:"actor_id,omitempty"`
	Created   *google_protobuf.Timestamp `protobuf:"bytes,4,opt,name=created" json:"created,omitempty"`
	// label is populated for "labeled" and "unlabeled" events.
	// The label will usually not have an ID, due to Github's API
	// not returning one.
	Label *GithubLabel `protobuf:"bytes,5,opt,name=label" json:"label,omitempty"`
	// milestone is populated for "milestoned" and "demilestoned" events.
	// The label will usually not have an ID, due to Github's API
	// not returning one.
	Milestone *GithubMilestone `protobuf:"bytes,6,opt,name=milestone" json:"milestone,omitempty"`
	// For "assigned", "unassigned":
	AssigneeId int64 `protobuf:"varint,7,opt,name=assignee_id,json=assigneeId" json:"assignee_id,omitempty"`
	AssignerId int64 `protobuf:"varint,8,opt,name=assigner_id,json=assignerId" json:"assigner_id,omitempty"`
	// For "referenced", "closed":
	Commit *GithubCommit `protobuf:"bytes,9,opt,name=commit" json:"commit,omitempty"`
	// For "renamed" events:
	RenameFrom string `protobuf:"bytes,11,opt,name=rename_from,json=renameFrom" json:"rename_from,omitempty"`
	RenameTo   string `protobuf:"bytes,12,opt,name=rename_to,json=renameTo" json:"rename_to,omitempty"`
	// other_json is usually empty. If Github adds event types or fields
	// in the future, this captures those added fields. If non-empty it
	// will be a JSON object with the fields that weren't understood.
	OtherJson []byte `protobuf:"bytes,10,opt,name=other_json,json=otherJson,proto3" json:"other_json,omitempty"`
}

func (m *GithubIssueEvent) Reset()                    { *m = GithubIssueEvent{} }
func (m *GithubIssueEvent) String() string            { return proto.CompactTextString(m) }
func (*GithubIssueEvent) ProtoMessage()               {}
func (*GithubIssueEvent) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{6} }

func (m *GithubIssueEvent) GetId() int64 {
	if m != nil {
		return m.Id
	}
	return 0
}

func (m *GithubIssueEvent) GetEventType() string {
	if m != nil {
		return m.EventType
	}
	return ""
}

func (m *GithubIssueEvent) GetActorId() int64 {
	if m != nil {
		return m.ActorId
	}
	return 0
}

func (m *GithubIssueEvent) GetCreated() *google_protobuf.Timestamp {
	if m != nil {
		return m.Created
	}
	return nil
}

func (m *GithubIssueEvent) GetLabel() *GithubLabel {
	if m != nil {
		return m.Label
	}
	return nil
}

func (m *GithubIssueEvent) GetMilestone() *GithubMilestone {
	if m != nil {
		return m.Milestone
	}
	return nil
}

func (m *GithubIssueEvent) GetAssigneeId() int64 {
	if m != nil {
		return m.AssigneeId
	}
	return 0
}

func (m *GithubIssueEvent) GetAssignerId() int64 {
	if m != nil {
		return m.AssignerId
	}
	return 0
}

func (m *GithubIssueEvent) GetCommit() *GithubCommit {
	if m != nil {
		return m.Commit
	}
	return nil
}

func (m *GithubIssueEvent) GetRenameFrom() string {
	if m != nil {
		return m.RenameFrom
	}
	return ""
}

func (m *GithubIssueEvent) GetRenameTo() string {
	if m != nil {
		return m.RenameTo
	}
	return ""
}

func (m *GithubIssueEvent) GetOtherJson() []byte {
	if m != nil {
		return m.OtherJson
	}
	return nil
}

type GithubCommit struct {
	Owner    string `protobuf:"bytes,1,opt,name=owner" json:"owner,omitempty"`
	Repo     string `protobuf:"bytes,2,opt,name=repo" json:"repo,omitempty"`
	CommitId string `protobuf:"bytes,3,opt,name=commit_id,json=commitId" json:"commit_id,omitempty"`
}

func (m *GithubCommit) Reset()                    { *m = GithubCommit{} }
func (m *GithubCommit) String() string            { return proto.CompactTextString(m) }
func (*GithubCommit) ProtoMessage()               {}
func (*GithubCommit) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{7} }

func (m *GithubCommit) GetOwner() string {
	if m != nil {
		return m.Owner
	}
	return ""
}

func (m *GithubCommit) GetRepo() string {
	if m != nil {
		return m.Repo
	}
	return ""
}

func (m *GithubCommit) GetCommitId() string {
	if m != nil {
		return m.CommitId
	}
	return ""
}

// GithubIssueSyncStatus notes where syncing is at for comments
// on an issue,
// This mutation type is only made at/after the same top-level mutation
// which created the corresponding comments.
type GithubIssueSyncStatus struct {
	// server_date is the "Date" response header from Github for the
	// final HTTP response.
	ServerDate *google_protobuf.Timestamp `protobuf:"bytes,1,opt,name=server_date,json=serverDate" json:"server_date,omitempty"`
}

func (m *GithubIssueSyncStatus) Reset()                    { *m = GithubIssueSyncStatus{} }
func (m *GithubIssueSyncStatus) String() string            { return proto.CompactTextString(m) }
func (*GithubIssueSyncStatus) ProtoMessage()               {}
func (*GithubIssueSyncStatus) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{8} }

func (m *GithubIssueSyncStatus) GetServerDate() *google_protobuf.Timestamp {
	if m != nil {
		return m.ServerDate
	}
	return nil
}

type GithubIssueCommentMutation struct {
	Id      int64                      `protobuf:"varint,1,opt,name=id" json:"id,omitempty"`
	User    *GithubUser                `protobuf:"bytes,2,opt,name=user" json:"user,omitempty"`
	Body    string                     `protobuf:"bytes,3,opt,name=body" json:"body,omitempty"`
	Created *google_protobuf.Timestamp `protobuf:"bytes,4,opt,name=created" json:"created,omitempty"`
	Updated *google_protobuf.Timestamp `protobuf:"bytes,5,opt,name=updated" json:"updated,omitempty"`
}

func (m *GithubIssueCommentMutation) Reset()                    { *m = GithubIssueCommentMutation{} }
func (m *GithubIssueCommentMutation) String() string            { return proto.CompactTextString(m) }
func (*GithubIssueCommentMutation) ProtoMessage()               {}
func (*GithubIssueCommentMutation) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{9} }

func (m *GithubIssueCommentMutation) GetId() int64 {
	if m != nil {
		return m.Id
	}
	return 0
}

func (m *GithubIssueCommentMutation) GetUser() *GithubUser {
	if m != nil {
		return m.User
	}
	return nil
}

func (m *GithubIssueCommentMutation) GetBody() string {
	if m != nil {
		return m.Body
	}
	return ""
}

func (m *GithubIssueCommentMutation) GetCreated() *google_protobuf.Timestamp {
	if m != nil {
		return m.Created
	}
	return nil
}

func (m *GithubIssueCommentMutation) GetUpdated() *google_protobuf.Timestamp {
	if m != nil {
		return m.Updated
	}
	return nil
}

type GithubUser struct {
	Id    int64  `protobuf:"varint,1,opt,name=id" json:"id,omitempty"`
	Login string `protobuf:"bytes,2,opt,name=login" json:"login,omitempty"`
}

func (m *GithubUser) Reset()                    { *m = GithubUser{} }
func (m *GithubUser) String() string            { return proto.CompactTextString(m) }
func (*GithubUser) ProtoMessage()               {}
func (*GithubUser) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{10} }

func (m *GithubUser) GetId() int64 {
	if m != nil {
		return m.Id
	}
	return 0
}

func (m *GithubUser) GetLogin() string {
	if m != nil {
		return m.Login
	}
	return ""
}

type GitMutation struct {
	Repo *GitRepo `protobuf:"bytes,1,opt,name=repo" json:"repo,omitempty"`
	// commit adds a commit, or adds new information to a commit if fields
	// are added in the future.
	Commit *GitCommit `protobuf:"bytes,2,opt,name=commit" json:"commit,omitempty"`
}

func (m *GitMutation) Reset()                    { *m = GitMutation{} }
func (m *GitMutation) String() string            { return proto.CompactTextString(m) }
func (*GitMutation) ProtoMessage()               {}
func (*GitMutation) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{11} }

func (m *GitMutation) GetRepo() *GitRepo {
	if m != nil {
		return m.Repo
	}
	return nil
}

func (m *GitMutation) GetCommit() *GitCommit {
	if m != nil {
		return m.Commit
	}
	return nil
}

// GitRepo identifies a git repo being mutated.
type GitRepo struct {
	// If go_repo is set, it identifies a go.googlesource.com/<go_repo> repo.
	GoRepo string `protobuf:"bytes,1,opt,name=go_repo,json=goRepo" json:"go_repo,omitempty"`
}

func (m *GitRepo) Reset()                    { *m = GitRepo{} }
func (m *GitRepo) String() string            { return proto.CompactTextString(m) }
func (*GitRepo) ProtoMessage()               {}
func (*GitRepo) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{12} }

func (m *GitRepo) GetGoRepo() string {
	if m != nil {
		return m.GoRepo
	}
	return ""
}

type GitCommit struct {
	Sha1 string `protobuf:"bytes,1,opt,name=sha1" json:"sha1,omitempty"`
	// raw is the "git cat-file commit $sha1" output.
	Raw      []byte       `protobuf:"bytes,2,opt,name=raw,proto3" json:"raw,omitempty"`
	DiffTree *GitDiffTree `protobuf:"bytes,3,opt,name=diff_tree,json=diffTree" json:"diff_tree,omitempty"`
}

func (m *GitCommit) Reset()                    { *m = GitCommit{} }
func (m *GitCommit) String() string            { return proto.CompactTextString(m) }
func (*GitCommit) ProtoMessage()               {}
func (*GitCommit) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{13} }

func (m *GitCommit) GetSha1() string {
	if m != nil {
		return m.Sha1
	}
	return ""
}

func (m *GitCommit) GetRaw() []byte {
	if m != nil {
		return m.Raw
	}
	return nil
}

func (m *GitCommit) GetDiffTree() *GitDiffTree {
	if m != nil {
		return m.DiffTree
	}
	return nil
}

// git diff-tree --numstat oldtree newtree
type GitDiffTree struct {
	File []*GitDiffTreeFile `protobuf:"bytes,1,rep,name=file" json:"file,omitempty"`
}

func (m *GitDiffTree) Reset()                    { *m = GitDiffTree{} }
func (m *GitDiffTree) String() string            { return proto.CompactTextString(m) }
func (*GitDiffTree) ProtoMessage()               {}
func (*GitDiffTree) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{14} }

func (m *GitDiffTree) GetFile() []*GitDiffTreeFile {
	if m != nil {
		return m.File
	}
	return nil
}

// GitDiffTreeFile represents one line of `git diff-tree --numstat` output.
type GitDiffTreeFile struct {
	File    string `protobuf:"bytes,1,opt,name=file" json:"file,omitempty"`
	Added   int64  `protobuf:"varint,2,opt,name=added" json:"added,omitempty"`
	Deleted int64  `protobuf:"varint,3,opt,name=deleted" json:"deleted,omitempty"`
	Binary  bool   `protobuf:"varint,4,opt,name=binary" json:"binary,omitempty"`
}

func (m *GitDiffTreeFile) Reset()                    { *m = GitDiffTreeFile{} }
func (m *GitDiffTreeFile) String() string            { return proto.CompactTextString(m) }
func (*GitDiffTreeFile) ProtoMessage()               {}
func (*GitDiffTreeFile) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{15} }

func (m *GitDiffTreeFile) GetFile() string {
	if m != nil {
		return m.File
	}
	return ""
}

func (m *GitDiffTreeFile) GetAdded() int64 {
	if m != nil {
		return m.Added
	}
	return 0
}

func (m *GitDiffTreeFile) GetDeleted() int64 {
	if m != nil {
		return m.Deleted
	}
	return 0
}

func (m *GitDiffTreeFile) GetBinary() bool {
	if m != nil {
		return m.Binary
	}
	return false
}

type GerritMutation struct {
	// Project is the Gerrit server and project, without scheme (https implied) or
	// trailing slash.
	Project string `protobuf:"bytes,1,opt,name=project" json:"project,omitempty"`
	// Commits to add.
	Commits []*GitCommit `protobuf:"bytes,2,rep,name=commits" json:"commits,omitempty"`
	// git refs to update.
	Refs []*GitRef `protobuf:"bytes,3,rep,name=refs" json:"refs,omitempty"`
}

func (m *GerritMutation) Reset()                    { *m = GerritMutation{} }
func (m *GerritMutation) String() string            { return proto.CompactTextString(m) }
func (*GerritMutation) ProtoMessage()               {}
func (*GerritMutation) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{16} }

func (m *GerritMutation) GetProject() string {
	if m != nil {
		return m.Project
	}
	return ""
}

func (m *GerritMutation) GetCommits() []*GitCommit {
	if m != nil {
		return m.Commits
	}
	return nil
}

func (m *GerritMutation) GetRefs() []*GitRef {
	if m != nil {
		return m.Refs
	}
	return nil
}

type GitRef struct {
	// ref is the git ref name, such as:
	//    HEAD
	//    refs/heads/master
	//    refs/changes/00/14700/1
	//    refs/changes/00/14700/meta
	//    refs/meta/config
	Ref string `protobuf:"bytes,1,opt,name=ref" json:"ref,omitempty"`
	// sha1 is the lowercase hex sha1
	Sha1 string `protobuf:"bytes,2,opt,name=sha1" json:"sha1,omitempty"`
}

func (m *GitRef) Reset()                    { *m = GitRef{} }
func (m *GitRef) String() string            { return proto.CompactTextString(m) }
func (*GitRef) ProtoMessage()               {}
func (*GitRef) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{17} }

func (m *GitRef) GetRef() string {
	if m != nil {
		return m.Ref
	}
	return ""
}

func (m *GitRef) GetSha1() string {
	if m != nil {
		return m.Sha1
	}
	return ""
}

func init() {
	proto.RegisterType((*Mutation)(nil), "maintpb.Mutation")
	proto.RegisterType((*GithubMutation)(nil), "maintpb.GithubMutation")
	proto.RegisterType((*GithubIssueMutation)(nil), "maintpb.GithubIssueMutation")
	proto.RegisterType((*BoolChange)(nil), "maintpb.BoolChange")
	proto.RegisterType((*GithubLabel)(nil), "maintpb.GithubLabel")
	proto.RegisterType((*GithubMilestone)(nil), "maintpb.GithubMilestone")
	proto.RegisterType((*GithubIssueEvent)(nil), "maintpb.GithubIssueEvent")
	proto.RegisterType((*GithubCommit)(nil), "maintpb.GithubCommit")
	proto.RegisterType((*GithubIssueSyncStatus)(nil), "maintpb.GithubIssueSyncStatus")
	proto.RegisterType((*GithubIssueCommentMutation)(nil), "maintpb.GithubIssueCommentMutation")
	proto.RegisterType((*GithubUser)(nil), "maintpb.GithubUser")
	proto.RegisterType((*GitMutation)(nil), "maintpb.GitMutation")
	proto.RegisterType((*GitRepo)(nil), "maintpb.GitRepo")
	proto.RegisterType((*GitCommit)(nil), "maintpb.GitCommit")
	proto.RegisterType((*GitDiffTree)(nil), "maintpb.GitDiffTree")
	proto.RegisterType((*GitDiffTreeFile)(nil), "maintpb.GitDiffTreeFile")
	proto.RegisterType((*GerritMutation)(nil), "maintpb.GerritMutation")
	proto.RegisterType((*GitRef)(nil), "maintpb.GitRef")
}

func init() { proto.RegisterFile("maintner.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 1251 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x9c, 0x56, 0xfd, 0x8e, 0xdb, 0x44,
	0x10, 0x57, 0xe2, 0x7c, 0xd8, 0x93, 0x34, 0x77, 0xdd, 0x7e, 0x6d, 0xaf, 0xa5, 0x1c, 0x2e, 0xa2,
	0xa7, 0xb6, 0xe4, 0x68, 0x41, 0x80, 0x54, 0x21, 0xd4, 0x4f, 0x74, 0x08, 0xf8, 0x63, 0x7b, 0xfd,
	0xdb, 0x72, 0xe2, 0x8d, 0xcf, 0xc5, 0xf6, 0x86, 0xf5, 0xfa, 0x4a, 0x24, 0x78, 0x19, 0x9e, 0x85,
	0x57, 0xe0, 0x39, 0x78, 0x05, 0xb4, 0xb3, 0xeb, 0x8f, 0xe4, 0xd2, 0x5e, 0xcb, 0x7f, 0x33, 0xb3,
	0xbf, 0x99, 0x9d, 0xaf, 0x9d, 0x59, 0x98, 0x64, 0x61, 0x92, 0xab, 0x9c, 0xcb, 0xe9, 0x52, 0x0a,
	0x25, 0xc8, 0x10, 0xf9, 0xe5, 0x6c, 0xef, 0x51, 0x9c, 0xa8, 0x93, 0x72, 0x36, 0x9d, 0x8b, 0xec,
	0x30, 0x16, 0x69, 0x98, 0xc7, 0x87, 0x88, 0x98, 0x95, 0x8b, 0xc3, 0xa5, 0x5a, 0x2d, 0x79, 0x71,
	0xa8, 0x92, 0x8c, 0x17, 0x2a, 0xcc, 0x96, 0x0d, 0x65, 0xac, 0xf8, 0xff, 0x74, 0xc0, 0xfd, 0xb9,
	0x54, 0xa1, 0x4a, 0x44, 0x4e, 0xbe, 0x87, 0xb1, 0xb1, 0x15, 0x24, 0x45, 0x51, 0x72, 0xda, 0xd9,
	0xef, 0x1c, 0x8c, 0x1e, 0xde, 0x9c, 0xda, 0x9b, 0xa6, 0x3f, 0xe0, 0xe1, 0x91, 0x3e, 0xab, 0x74,
	0xd8, 0x28, 0x6e, 0x84, 0xe4, 0x10, 0x06, 0x86, 0xa5, 0x0e, 0xaa, 0x5e, 0xdb, 0x50, 0xad, 0xb5,
	0x2c, 0x8c, 0x7c, 0x06, 0x4e, 0x9c, 0x28, 0xda, 0x45, 0xf4, 0xe5, 0x36, 0xba, 0x86, 0x6a, 0x00,
	0x1a, 0xe6, 0x52, 0x26, 0x8a, 0xf6, 0x36, 0x0d, 0xa3, 0xb8, 0x65, 0x18, 0x79, 0xff, 0xaf, 0x0e,
	0x4c, 0xd6, 0xef, 0x24, 0x97, 0xa1, 0x2f, 0xde, 0xe4, 0x5c, 0x62, 0x58, 0x1e, 0x33, 0x0c, 0x21,
	0xd0, 0x93, 0x7c, 0x29, 0xd0, 0x05, 0x8f, 0x21, 0x4d, 0xee, 0xc3, 0x20, 0x0d, 0x67, 0x3c, 0x2d,
	0xa8, 0xb3, 0xef, 0x6c, 0x3a, 0x76, 0x52, 0xce, 0x7e, 0xd2, 0x87, 0xcc, 0x62, 0xc8, 0xb7, 0x00,
	0x59, 0x92, 0xf2, 0x42, 0x89, 0x9c, 0x17, 0xb4, 0x87, 0x1a, 0x74, 0x33, 0xf0, 0x0a, 0xc0, 0x5a,
	0x58, 0xff, 0x5f, 0x17, 0x2e, 0x6d, 0xc9, 0xe9, 0x07, 0x78, 0x7a, 0x15, 0x06, 0x79, 0x99, 0xcd,
	0xb8, 0xc4, 0x84, 0xf7, 0x99, 0xe5, 0xc8, 0x0d, 0xf0, 0x72, 0xa1, 0x02, 0xfe, 0x7b, 0x52, 0x28,
	0x7a, 0x61, 0xbf, 0x73, 0xe0, 0x32, 0x37, 0x17, 0xea, 0xb9, 0xe6, 0xc9, 0x04, 0xba, 0x49, 0x44,
	0xc7, 0xfb, 0x9d, 0x03, 0x87, 0x75, 0x93, 0x88, 0xdc, 0x81, 0x5e, 0x59, 0x70, 0x69, 0x53, 0x7b,
	0x69, 0xc3, 0xf5, 0x57, 0x05, 0x97, 0x0c, 0x01, 0xe4, 0x01, 0x78, 0x61, 0x51, 0x24, 0x71, 0xce,
	0x79, 0x41, 0x01, 0x03, 0xdd, 0x8a, 0x6e, 0x50, 0xe4, 0x1e, 0x5c, 0x8c, 0x78, 0xca, 0x15, 0x8f,
	0x82, 0x46, 0x75, 0xb4, 0xef, 0x1c, 0x38, 0x6c, 0xd7, 0x1e, 0x3c, 0xae, 0xc1, 0x5f, 0xc1, 0x70,
	0x2e, 0x79, 0xa8, 0x78, 0x44, 0xfb, 0xe8, 0xcb, 0xde, 0x34, 0x16, 0x22, 0x4e, 0xf9, 0xb4, 0x6a,
	0xe8, 0xe9, 0x71, 0xd5, 0xbf, 0xac, 0x82, 0x6a, 0xad, 0x72, 0x19, 0xa1, 0xd6, 0xe0, 0x7c, 0x2d,
	0x0b, 0xd5, 0xd9, 0x9c, 0x89, 0x68, 0x45, 0x87, 0x26, 0x9b, 0x9a, 0xd6, 0x79, 0x57, 0x89, 0x4a,
	0x39, 0xf5, 0x4c, 0xde, 0x91, 0x21, 0x9f, 0xc0, 0x38, 0x17, 0x41, 0x5d, 0x36, 0xba, 0x83, 0xe9,
	0x1c, 0xe5, 0xa2, 0x2e, 0xaa, 0x86, 0xd4, 0xe7, 0x41, 0x12, 0xd1, 0x5d, 0xcc, 0xed, 0xa8, 0x96,
	0x1d, 0x45, 0xe4, 0x36, 0x5c, 0x68, 0x20, 0x79, 0x99, 0xd1, 0x8b, 0x88, 0x69, 0xf4, 0x7e, 0x29,
	0x33, 0x72, 0x07, 0x76, 0x1a, 0x90, 0x71, 0x85, 0xa0, 0x2b, 0x93, 0x5a, 0x7c, 0x8c, 0x3e, 0xdd,
	0x83, 0xc1, 0x3c, 0x15, 0x05, 0x8f, 0xe8, 0xa5, 0x8d, 0xa2, 0x3d, 0x11, 0x22, 0x7d, 0x7a, 0x12,
	0xe6, 0x31, 0x67, 0x16, 0xa2, 0xc1, 0xa9, 0x98, 0xff, 0xca, 0x23, 0x7a, 0xfd, 0x1d, 0x60, 0x03,
	0xd1, 0xa1, 0x2c, 0xcb, 0x34, 0x0d, 0x24, 0xff, 0xad, 0xe4, 0x85, 0xa2, 0x37, 0x4d, 0xb4, 0x5a,
	0xc6, 0x8c, 0x88, 0x7c, 0x03, 0x9e, 0xb1, 0x1c, 0x84, 0x8a, 0x5e, 0x39, 0x37, 0xe5, 0xae, 0x01,
	0x3f, 0x56, 0xe4, 0x8b, 0x5a, 0x71, 0xb6, 0xa2, 0x57, 0xdf, 0xde, 0x6d, 0x56, 0xe3, 0xc9, 0x4a,
	0x7b, 0x23, 0x79, 0x26, 0x4e, 0x79, 0x80, 0x8f, 0x8d, 0x5e, 0xc3, 0xce, 0x19, 0x19, 0x19, 0x3e,
	0x43, 0x6c, 0xca, 0x28, 0xb2, 0xe7, 0xf4, 0x1d, 0xef, 0xd5, 0x0d, 0xa3, 0xc8, 0xa8, 0x7c, 0x07,
	0xc3, 0xb9, 0xc8, 0x32, 0x9e, 0x2b, 0xea, 0xa2, 0xc2, 0xed, 0x6d, 0x23, 0xee, 0xa9, 0x81, 0xd4,
	0xa3, 0xa5, 0xd2, 0x21, 0xcf, 0x61, 0x62, 0xc9, 0xa0, 0x50, 0xa1, 0x2a, 0x0b, 0x3a, 0xc1, 0x58,
	0x6e, 0x6d, 0xb3, 0xf2, 0x72, 0x95, 0xcf, 0x5f, 0x22, 0x8a, 0x5d, 0xb0, 0x5a, 0x86, 0x25, 0x87,
	0xd0, 0xe7, 0xa7, 0xda, 0x87, 0x3d, 0xf4, 0xe1, 0xfa, 0x36, 0xed, 0xe7, 0x1a, 0xc0, 0x0c, 0x8e,
	0x3c, 0x86, 0x31, 0x12, 0xd5, 0xad, 0x37, 0xde, 0xeb, 0xd6, 0x11, 0xea, 0x18, 0xc6, 0xbf, 0x05,
	0xd0, 0xd4, 0x9c, 0xec, 0x82, 0x73, 0x1a, 0xa6, 0x38, 0x65, 0x5c, 0xa6, 0x49, 0xff, 0x01, 0x8c,
	0x5a, 0x29, 0xb3, 0x93, 0xa2, 0x53, 0x4f, 0x0a, 0x02, 0xbd, 0x3c, 0xcc, 0x78, 0x35, 0x82, 0x34,
	0xed, 0xff, 0x01, 0x3b, 0x1b, 0x33, 0xee, 0x8c, 0x5a, 0xfd, 0xae, 0xba, 0xed, 0x77, 0xd5, 0xf4,
	0xb0, 0x73, 0x7e, 0x0f, 0x37, 0x83, 0xae, 0x87, 0x66, 0x2d, 0xe7, 0xff, 0xed, 0xc0, 0xee, 0x66,
	0xbe, 0xce, 0xdc, 0xff, 0x11, 0x80, 0x49, 0x9c, 0xde, 0x86, 0xd6, 0x09, 0x0f, 0x25, 0xc7, 0xab,
	0x25, 0x27, 0xd7, 0xc1, 0x0d, 0xe7, 0x4a, 0x48, 0xfd, 0x72, 0x1d, 0x54, 0x1a, 0x22, 0x7f, 0x14,
	0xb5, 0x27, 0x52, 0xef, 0xfd, 0x27, 0xd2, 0x5d, 0xe8, 0x9b, 0x76, 0xec, 0x9f, 0xdd, 0x6b, 0x75,
	0x3b, 0x1a, 0x08, 0xf9, 0x1a, 0xbc, 0x66, 0xb4, 0x98, 0xf9, 0xf5, 0xf6, 0xe5, 0xd1, 0x40, 0xc9,
	0xc7, 0x30, 0xaa, 0x06, 0xaa, 0xf6, 0x7b, 0x88, 0x7e, 0x43, 0x25, 0x3a, 0x8a, 0x5a, 0x00, 0x0c,
	0xcc, 0x5d, 0x03, 0xe8, 0xd8, 0x3e, 0x87, 0x81, 0x6e, 0xc8, 0x44, 0xe1, 0xb8, 0x1b, 0x3d, 0xbc,
	0xb2, 0x71, 0xed, 0x53, 0x3c, 0x64, 0x16, 0xa4, 0xed, 0x49, 0xae, 0x2b, 0x1e, 0x2c, 0xa4, 0xc8,
	0xe8, 0x08, 0xb3, 0x08, 0x46, 0xf4, 0x42, 0x8a, 0x4c, 0xef, 0x1c, 0x0b, 0x50, 0x02, 0xb7, 0x8b,
	0xc7, 0x5c, 0x23, 0x38, 0x16, 0xba, 0x04, 0x42, 0x9d, 0x70, 0x19, 0xbc, 0x2e, 0x44, 0x4e, 0x61,
	0xbf, 0x73, 0x30, 0x66, 0x1e, 0x4a, 0x7e, 0x2c, 0x44, 0xee, 0xbf, 0x82, 0x71, 0xfb, 0xd2, 0x0f,
	0xd8, 0x80, 0x37, 0xc0, 0x33, 0x0e, 0x56, 0xd5, 0xf3, 0x98, 0x6b, 0x04, 0x47, 0x91, 0x7f, 0x0c,
	0x57, 0xb6, 0x3e, 0x0a, 0xf2, 0x08, 0x46, 0x05, 0x97, 0xa7, 0x5c, 0x06, 0x7a, 0x1b, 0xd8, 0x8f,
	0xce, 0xbb, 0x6a, 0x0b, 0x06, 0xfe, 0x2c, 0x54, 0x5c, 0xff, 0x99, 0xf6, 0xde, 0x3e, 0x27, 0xce,
	0x74, 0x5f, 0xb5, 0x5e, 0xbb, 0xe7, 0xad, 0xd7, 0x6a, 0x25, 0x39, 0xad, 0x95, 0xf4, 0xff, 0x1a,
	0xb0, 0xb5, 0x12, 0xfb, 0xef, 0xbd, 0x12, 0xfd, 0x87, 0x00, 0x8d, 0x4f, 0xdb, 0x1e, 0x71, 0x2a,
	0xe2, 0x24, 0xaf, 0x1e, 0x31, 0x32, 0x7e, 0x80, 0x03, 0xa3, 0x8e, 0xfd, 0x53, 0x5b, 0x21, 0x93,
	0xd0, 0xdd, 0x76, 0xac, 0x8c, 0x2f, 0x85, 0xad, 0xd9, 0xdd, 0xba, 0xf3, 0x4c, 0x4e, 0x48, 0x1b,
	0xb7, 0xde, 0x76, 0xbe, 0x0f, 0x43, 0xab, 0x4c, 0xae, 0xc1, 0x30, 0x16, 0x41, 0x6d, 0xdf, 0x63,
	0x83, 0x58, 0xe8, 0x03, 0x3f, 0x02, 0xaf, 0x56, 0xd4, 0x59, 0x2c, 0x4e, 0xc2, 0x07, 0x16, 0x82,
	0xb4, 0x1e, 0x74, 0x32, 0x7c, 0x83, 0xb7, 0x8d, 0x99, 0x26, 0xf5, 0xd6, 0x88, 0x92, 0xc5, 0x22,
	0x50, 0x92, 0x73, 0x3b, 0x7f, 0xd6, 0x9e, 0xe9, 0xb3, 0x64, 0xb1, 0x38, 0x96, 0x9c, 0x33, 0x37,
	0xb2, 0x94, 0xff, 0x08, 0x43, 0xad, 0x0e, 0xc8, 0x7d, 0xe8, 0x2d, 0x92, 0x54, 0xf7, 0xce, 0x99,
	0x0f, 0x5f, 0x85, 0x79, 0x91, 0xa4, 0x9c, 0x21, 0xca, 0xcf, 0x70, 0x4a, 0xb6, 0x0f, 0xb4, 0xa3,
	0xd6, 0x00, 0x3a, 0xaa, 0x69, 0x9d, 0xe4, 0x30, 0x8a, 0x78, 0x84, 0xae, 0x3a, 0xcc, 0x30, 0x84,
	0xc2, 0xd0, 0xfe, 0x95, 0xaa, 0xf9, 0x64, 0x59, 0x3d, 0x16, 0x67, 0x49, 0x1e, 0xca, 0x15, 0x76,
	0x87, 0xcb, 0x2c, 0xe7, 0xff, 0x09, 0x93, 0xf5, 0x8f, 0xb1, 0xb6, 0xb1, 0x94, 0xe2, 0x35, 0x9f,
	0x2b, 0x7b, 0x61, 0xc5, 0x92, 0xfb, 0x66, 0x1b, 0x26, 0xaa, 0xa0, 0x5d, 0x8c, 0x65, 0x5b, 0x39,
	0x2a, 0x08, 0xb9, 0xad, 0x2b, 0xbc, 0xa8, 0x7e, 0xc6, 0x3b, 0xeb, 0x15, 0x5e, 0x30, 0x3c, 0xf4,
	0xa7, 0x30, 0x30, 0x3c, 0x66, 0x9e, 0x2f, 0xec, 0x95, 0x9a, 0xac, 0xeb, 0xd3, 0x6d, 0xea, 0x33,
	0x1b, 0x60, 0x5b, 0x7e, 0xf9, 0x5f, 0x00, 0x00, 0x00, 0xff, 0xff, 0x70, 0x1d, 0x68, 0x88, 0xe4,
	0x0c, 0x00, 0x00,
}
