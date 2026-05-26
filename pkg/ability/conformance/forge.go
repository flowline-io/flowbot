package conformance

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/ability/forge"
)

// ForgeConfig configures the fake backend for each forge conformance subtest.
type ForgeConfig struct {
	User           *ability.ForgeUser
	UserErr        error
	Repo           *ability.ForgeRepo
	RepoErr        error
	Issues         []*ability.ForgeIssue
	IssuesErr      error
	Issue          *ability.ForgeIssue
	IssueErr       error
	Diff           *ability.ForgeCommitDiff
	DiffErr        error
	FileContent    []byte
	FileContentErr error
}

// ForgeServiceFactory creates a fresh forge Service wired to a fake backend.
type ForgeServiceFactory func(t *testing.T, cfg ForgeConfig) forge.Service

// RunForgeConformance runs the standard forge capability conformance suite.
func RunForgeConformance(t *testing.T, factory ForgeServiceFactory) {
	t.Run("get user success", func(t *testing.T) {
		svc := factory(t, ForgeConfig{
			User: &ability.ForgeUser{ID: 1, UserName: "testuser", Email: "test@example.com"},
		})
		user, err := svc.GetUser(t.Context())
		require.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, int64(1), user.ID)
		assert.Equal(t, "testuser", user.UserName)
	})

	t.Run("get user timeout", func(t *testing.T) {
		svc := factory(t, ForgeConfig{})
		_, err := svc.GetUser(CanceledContext())
		RequireTimeoutError(t, err)
	})

	t.Run("get user provider error", func(t *testing.T) {
		svc := factory(t, ForgeConfig{UserErr: assert.AnError})
		_, err := svc.GetUser(t.Context())
		RequireProviderError(t, err)
	})

	t.Run("get repo success", func(t *testing.T) {
		svc := factory(t, ForgeConfig{
			Repo: &ability.ForgeRepo{ID: 1, Name: "repo", FullName: "owner/repo", Owner: "owner"},
		})
		repo, err := svc.GetRepo(t.Context(), "owner", "repo")
		require.NoError(t, err)
		require.NotNil(t, repo)
		assert.Equal(t, "repo", repo.Name)
		assert.Equal(t, "owner/repo", repo.FullName)
	})

	t.Run("get repo timeout", func(t *testing.T) {
		svc := factory(t, ForgeConfig{})
		_, err := svc.GetRepo(CanceledContext(), "", "")
		RequireTimeoutError(t, err)
	})

	t.Run("get repo empty owner", func(t *testing.T) {
		svc := factory(t, ForgeConfig{})
		_, err := svc.GetRepo(t.Context(), "", "repo")
		RequireInvalidArgError(t, err)
	})

	t.Run("get repo empty name", func(t *testing.T) {
		svc := factory(t, ForgeConfig{})
		_, err := svc.GetRepo(t.Context(), "owner", "")
		RequireInvalidArgError(t, err)
	})

	t.Run("get repo provider error", func(t *testing.T) {
		svc := factory(t, ForgeConfig{RepoErr: assert.AnError})
		_, err := svc.GetRepo(t.Context(), "owner", "repo")
		RequireProviderError(t, err)
	})

	t.Run("list issues success", func(t *testing.T) {
		svc := factory(t, ForgeConfig{
			Issues: []*ability.ForgeIssue{
				{ID: 100, Index: 1, Title: "First issue", State: "open"},
				{ID: 200, Index: 2, Title: "Second issue", State: "closed"},
			},
		})
		result, err := svc.ListIssues(t.Context(), "owner", &forge.ListIssuesQuery{Page: ability.PageRequest{Limit: 20}})
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Items)
		require.NotNil(t, result.Page)
		assert.Len(t, result.Items, 2)
	})

	t.Run("list issues empty", func(t *testing.T) {
		svc := factory(t, ForgeConfig{})
		result, err := svc.ListIssues(t.Context(), "owner", &forge.ListIssuesQuery{})
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Items)
		assert.Empty(t, result.Items)
	})

	t.Run("list issues nil query", func(t *testing.T) {
		svc := factory(t, ForgeConfig{
			Issues: []*ability.ForgeIssue{{ID: 100, Index: 1, Title: "test", State: "open"}},
		})
		result, err := svc.ListIssues(t.Context(), "owner", nil)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("list issues timeout", func(t *testing.T) {
		svc := factory(t, ForgeConfig{})
		_, err := svc.ListIssues(CanceledContext(), "", nil)
		RequireTimeoutError(t, err)
	})

	t.Run("list issues empty owner", func(t *testing.T) {
		svc := factory(t, ForgeConfig{})
		_, err := svc.ListIssues(t.Context(), "", nil)
		RequireInvalidArgError(t, err)
	})

	t.Run("list issues provider error", func(t *testing.T) {
		svc := factory(t, ForgeConfig{IssuesErr: assert.AnError})
		_, err := svc.ListIssues(t.Context(), "owner", &forge.ListIssuesQuery{})
		RequireProviderError(t, err)
	})

	t.Run("get issue success", func(t *testing.T) {
		svc := factory(t, ForgeConfig{
			Issues: []*ability.ForgeIssue{{ID: 100, Index: 1, Title: "Test", State: "open"}},
		})
		issue, err := svc.GetIssue(t.Context(), "owner", "repo", 1)
		require.NoError(t, err)
		require.NotNil(t, issue)
		assert.Equal(t, int64(1), issue.Index)
	})

	t.Run("get issue timeout", func(t *testing.T) {
		svc := factory(t, ForgeConfig{})
		_, err := svc.GetIssue(CanceledContext(), "", "", 0)
		RequireTimeoutError(t, err)
	})

	t.Run("get issue empty owner", func(t *testing.T) {
		svc := factory(t, ForgeConfig{})
		_, err := svc.GetIssue(t.Context(), "", "repo", 1)
		RequireInvalidArgError(t, err)
	})

	t.Run("get issue provider error", func(t *testing.T) {
		svc := factory(t, ForgeConfig{IssuesErr: assert.AnError})
		_, err := svc.GetIssue(t.Context(), "owner", "repo", 1)
		RequireProviderError(t, err)
	})

	t.Run("get commit diff success", func(t *testing.T) {
		svc := factory(t, ForgeConfig{
			Diff: &ability.ForgeCommitDiff{
				CommitID:      "abc123",
				CommitMessage: "test commit",
				Files:         []string{"main.go"},
				DiffContent:   "diff content",
			},
		})
		diff, err := svc.GetCommitDiff(t.Context(), "owner", "repo", "abc123")
		require.NoError(t, err)
		require.NotNil(t, diff)
		assert.Equal(t, "abc123", diff.CommitID)
	})

	t.Run("get commit diff timeout", func(t *testing.T) {
		svc := factory(t, ForgeConfig{})
		_, err := svc.GetCommitDiff(CanceledContext(), "", "", "")
		RequireTimeoutError(t, err)
	})

	t.Run("get commit diff empty owner", func(t *testing.T) {
		svc := factory(t, ForgeConfig{})
		_, err := svc.GetCommitDiff(t.Context(), "", "repo", "abc123")
		RequireInvalidArgError(t, err)
	})

	t.Run("get commit diff empty commit id", func(t *testing.T) {
		svc := factory(t, ForgeConfig{})
		_, err := svc.GetCommitDiff(t.Context(), "owner", "repo", "")
		RequireInvalidArgError(t, err)
	})

	t.Run("get commit diff provider error", func(t *testing.T) {
		svc := factory(t, ForgeConfig{DiffErr: assert.AnError})
		_, err := svc.GetCommitDiff(t.Context(), "owner", "repo", "abc123")
		RequireProviderError(t, err)
	})

	t.Run("get file content success", func(t *testing.T) {
		svc := factory(t, ForgeConfig{
			FileContent: []byte("package main"),
		})
		content, err := svc.GetFileContent(t.Context(), "owner", "repo", "abc123", "main.go", 0, 0)
		require.NoError(t, err)
		assert.Equal(t, []byte("package main"), content)
	})

	t.Run("get file content timeout", func(t *testing.T) {
		svc := factory(t, ForgeConfig{})
		_, err := svc.GetFileContent(CanceledContext(), "", "", "", "", 0, 0)
		RequireTimeoutError(t, err)
	})

	t.Run("get file content empty owner", func(t *testing.T) {
		svc := factory(t, ForgeConfig{})
		_, err := svc.GetFileContent(t.Context(), "", "repo", "abc123", "main.go", 0, 0)
		RequireInvalidArgError(t, err)
	})

	t.Run("get file content empty file path", func(t *testing.T) {
		svc := factory(t, ForgeConfig{})
		_, err := svc.GetFileContent(t.Context(), "owner", "repo", "abc123", "", 0, 0)
		RequireInvalidArgError(t, err)
	})

	t.Run("get file content provider error", func(t *testing.T) {
		svc := factory(t, ForgeConfig{FileContentErr: assert.AnError})
		_, err := svc.GetFileContent(t.Context(), "owner", "repo", "abc123", "main.go", 0, 0)
		RequireProviderError(t, err)
	})
}
