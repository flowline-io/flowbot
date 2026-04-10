package gitea

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHookIssueAction_Constants(t *testing.T) {
	tests := []struct {
		action   HookIssueAction
		expected string
	}{
		{HookIssueOpened, "opened"},
		{HookIssueClosed, "closed"},
		{HookIssueReOpened, "reopened"},
		{HookIssueEdited, "edited"},
		{HookIssueAssigned, "assigned"},
		{HookIssueUnassigned, "unassigned"},
		{HookIssueLabelUpdated, "label_updated"},
		{HookIssueLabelCleared, "label_cleared"},
		{HookIssueSynchronized, "synchronized"},
		{HookIssueMilestoned, "milestoned"},
		{HookIssueDemilestoned, "demilestoned"},
		{HookIssueReviewed, "reviewed"},
		{HookIssueReviewRequested, "review_requested"},
		{HookIssueReviewRequestRemoved, "review_request_removed"},
		{HookIssueCreated, "created"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.action))
		})
	}
}

func TestConstants(t *testing.T) {
	assert.Equal(t, "gitea", ID)
	assert.Equal(t, "endpoint", EndpointKey)
	assert.Equal(t, "token", TokenKey)
}

func TestIssuePayload_JSONPayload(t *testing.T) {
	payload := &IssuePayload{
		Action:   HookIssueOpened,
		Index:    42,
		CommitID: "abc123",
	}

	data, err := payload.JSONPayload()
	assert.NoError(t, err)
	assert.NotNil(t, data)
	assert.Contains(t, string(data), `"action": "opened"`)
	assert.Contains(t, string(data), `"number": 42`)
}

func TestIssuePayload_WithChanges(t *testing.T) {
	payload := &IssuePayload{
		Action: HookIssueEdited,
		Index:  1,
		Changes: &ChangesPayload{
			Title: &ChangesFromPayload{
				From: "Old Title",
			},
			Body: &ChangesFromPayload{
				From: "Old Body",
			},
		},
	}

	data, err := payload.JSONPayload()
	assert.NoError(t, err)
	assert.NotNil(t, data)
	assert.Contains(t, string(data), `"title"`)
	assert.Contains(t, string(data), `"body"`)
}

func TestCommit(t *testing.T) {
	commit := Commit{
		Id:        "abc123def456",
		Message:   "Test commit message",
		Url:       "https://gitea.example.com/test/repo/commit/abc123",
		Timestamp: "2024-01-01T00:00:00Z",
		Added:     []string{"file1.go", "file2.go"},
		Removed:   []string{"old_file.go"},
		Modified:  []string{"modified.go"},
	}

	assert.Equal(t, "abc123def456", commit.Id)
	assert.Equal(t, "Test commit message", commit.Message)
	assert.Len(t, commit.Added, 2)
	assert.Len(t, commit.Removed, 1)
	assert.Len(t, commit.Modified, 1)
}

func TestCommitDiff(t *testing.T) {
	diff := CommitDiff{
		CommitID:      "abc123",
		CommitMessage: "Test commit",
		Files:         []string{"file1.go", "file2.go"},
		DiffContent:   "diff --git a/file1.go b/file1.go",
	}

	assert.Equal(t, "abc123", diff.CommitID)
	assert.Equal(t, "Test commit", diff.CommitMessage)
	assert.Len(t, diff.Files, 2)
	assert.NotEmpty(t, diff.DiffContent)
}

func TestChangesPayload(t *testing.T) {
	changes := &ChangesPayload{
		Title: &ChangesFromPayload{From: "Old Title"},
		Body:  &ChangesFromPayload{From: "Old Body"},
		Ref:   &ChangesFromPayload{From: "refs/heads/old-branch"},
	}

	assert.Equal(t, "Old Title", changes.Title.From)
	assert.Equal(t, "Old Body", changes.Body.From)
	assert.Equal(t, "refs/heads/old-branch", changes.Ref.From)
}

func TestChangesFromPayload(t *testing.T) {
	change := ChangesFromPayload{From: "previous value"}
	assert.Equal(t, "previous value", change.From)
}

func TestRepoPayload(t *testing.T) {
	payload := RepoPayload{
		Ref:          "refs/heads/main",
		Before:       "abc123",
		After:        "def456",
		CompareUrl:   "https://gitea.example.com/test/repo/compare/abc123...def456",
		Commits:      []*Commit{},
		TotalCommits: 0,
		HeadCommit:   nil,
	}

	assert.Equal(t, "refs/heads/main", payload.Ref)
	assert.Equal(t, "abc123", payload.Before)
	assert.Equal(t, "def456", payload.After)
	assert.NotEmpty(t, payload.CompareUrl)
}

func TestGitea_Constructor(t *testing.T) {
	// This test would need a mock Gitea server or test instance
	// For now, we just test the struct creation
	g := &Gitea{
		token: "test_token",
	}

	assert.Equal(t, "test_token", g.token)
	assert.Nil(t, g.c) // Client not initialized
}
