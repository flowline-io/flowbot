package gitea

import (
	"code.gitea.io/sdk/gitea"
	json "github.com/json-iterator/go"
)

// HookIssueAction represents the action that is sent along with an issue event.
type HookIssueAction string

const (
	// HookIssueOpened opened
	HookIssueOpened HookIssueAction = "opened"
	// HookIssueClosed closed
	HookIssueClosed HookIssueAction = "closed"
	// HookIssueReOpened reopened
	HookIssueReOpened HookIssueAction = "reopened"
	// HookIssueEdited edited
	HookIssueEdited HookIssueAction = "edited"
	// HookIssueAssigned assigned
	HookIssueAssigned HookIssueAction = "assigned"
	// HookIssueUnassigned unassigned
	HookIssueUnassigned HookIssueAction = "unassigned"
	// HookIssueLabelUpdated label_updated
	HookIssueLabelUpdated HookIssueAction = "label_updated"
	// HookIssueLabelCleared label_cleared
	HookIssueLabelCleared HookIssueAction = "label_cleared"
	// HookIssueSynchronized synchronized
	HookIssueSynchronized HookIssueAction = "synchronized"
	// HookIssueMilestoned is an issue action for when a milestone is set on an issue.
	HookIssueMilestoned HookIssueAction = "milestoned"
	// HookIssueDemilestoned is an issue action for when a milestone is cleared on an issue.
	HookIssueDemilestoned HookIssueAction = "demilestoned"
	// HookIssueReviewed is an issue action for when a pull request is reviewed
	HookIssueReviewed HookIssueAction = "reviewed"
	// HookIssueReviewRequested is an issue action for when a reviewer is requested for a pull request.
	HookIssueReviewRequested HookIssueAction = "review_requested"
	// HookIssueReviewRequestRemoved is an issue action for removing a review request to someone on a pull request.
	HookIssueReviewRequestRemoved HookIssueAction = "review_request_removed"
	// HookIssueCreated is an issue action for when an issue is created
	HookIssueCreated HookIssueAction = "created"
)

// IssuePayload represents the payload information that is sent along with an issue event.
type IssuePayload struct {
	Action     HookIssueAction   `json:"action"`
	Index      int64             `json:"number"`
	Changes    *ChangesPayload   `json:"changes,omitempty"`
	Issue      *gitea.Issue      `json:"issue"`
	Repository *gitea.Repository `json:"repository"`
	Sender     *gitea.User       `json:"sender"`
	CommitID   string            `json:"commit_id"`
}

// JSONPayload encodes the IssuePayload to JSON, with an indentation of two spaces.
func (p *IssuePayload) JSONPayload() ([]byte, error) {
	return json.MarshalIndent(p, "", "  ")
}

// ChangesFromPayload represents the payload information of issue change
type ChangesFromPayload struct {
	From string `json:"from"`
}

// ChangesPayload represents the payload information of issue change
type ChangesPayload struct {
	Title *ChangesFromPayload `json:"title,omitempty"`
	Body  *ChangesFromPayload `json:"body,omitempty"`
	Ref   *ChangesFromPayload `json:"ref,omitempty"`
}

type Commit struct {
	Id           string      `json:"id"`
	Message      string      `json:"message"`
	Url          string      `json:"url"`
	Author       *gitea.User `json:"author"`
	Committer    *gitea.User `json:"committer"`
	Verification any         `json:"verification"`
	Timestamp    string      `json:"timestamp"`
	Added        []string    `json:"added"`
	Removed      []string    `json:"removed"`
	Modified     []string    `json:"modified"`
}

type CommitDiff struct {
	CommitID      string   `json:"commit_id"`
	CommitMessage string   `json:"commit_message"`
	Files         []string `json:"files"`
	DiffContent   string   `json:"diff_content"`
}

type RepoPayload struct {
	Ref          string            `json:"ref"`
	Before       string            `json:"before"`
	After        string            `json:"after"`
	CompareUrl   string            `json:"compare_url"`
	Commits      []*Commit         `json:"commits"`
	TotalCommits int               `json:"total_commits"`
	HeadCommit   *Commit           `json:"head_commit"`
	Pusher       *gitea.User       `json:"pusher"`
	Repository   *gitea.Repository `json:"repository"`
	Sender       *gitea.User       `json:"sender"`
}
