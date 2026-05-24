package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/ability/conformance"
)

// Config holds mock backend behavior for conformance testing each Service method.
type Config struct {
	User            *ability.ForgeUser
	UserErr         error
	Repo            *ability.ForgeRepo
	RepoErr         error
	Issues          []*ability.ForgeIssue
	IssuesErr       error
	Diff            *ability.ForgeCommitDiff
	DiffErr         error
	FileContent     []byte
	FileContentErr  error
	Notifications   []*ability.Notification
	NotificationsErr error
	Releases        []*ability.Release
	ReleasesErr     error
}

// ServiceFactory creates a fresh Service wired to a fake backend.
type ServiceFactory func(t *testing.T, cfg Config) Service

// RunGithubConformance runs the standard GitHub capability conformance suite.
func RunGithubConformance(t *testing.T, factory ServiceFactory) {
	t.Helper()

	t.Run("get user success", func(t *testing.T) {
		svc := factory(t, Config{
			User: &ability.ForgeUser{ID: 1, UserName: "testuser", Email: "test@example.com"},
		})
		user, err := svc.GetUser(t.Context())
		require.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, int64(1), user.ID)
		assert.Equal(t, "testuser", user.UserName)
	})

	t.Run("get user timeout", func(t *testing.T) {
		svc := factory(t, Config{})
		_, err := svc.GetUser(conformance.CanceledContext())
		conformance.RequireTimeoutError(t, err)
	})

	t.Run("get user provider error", func(t *testing.T) {
		svc := factory(t, Config{UserErr: assert.AnError})
		_, err := svc.GetUser(t.Context())
		conformance.RequireProviderError(t, err)
	})

	t.Run("get repo success", func(t *testing.T) {
		svc := factory(t, Config{
			Repo: &ability.ForgeRepo{ID: 1, Name: "repo", FullName: "owner/repo", Owner: "owner"},
		})
		repo, err := svc.GetRepo(t.Context(), "owner", "repo")
		require.NoError(t, err)
		require.NotNil(t, repo)
		assert.Equal(t, "repo", repo.Name)
		assert.Equal(t, "owner/repo", repo.FullName)
	})

	t.Run("get repo timeout", func(t *testing.T) {
		svc := factory(t, Config{})
		_, err := svc.GetRepo(conformance.CanceledContext(), "", "")
		conformance.RequireTimeoutError(t, err)
	})

	t.Run("get repo empty owner", func(t *testing.T) {
		svc := factory(t, Config{})
		_, err := svc.GetRepo(t.Context(), "", "repo")
		conformance.RequireInvalidArgError(t, err)
	})

	t.Run("get repo empty name", func(t *testing.T) {
		svc := factory(t, Config{})
		_, err := svc.GetRepo(t.Context(), "owner", "")
		conformance.RequireInvalidArgError(t, err)
	})

	t.Run("get repo provider error", func(t *testing.T) {
		svc := factory(t, Config{RepoErr: assert.AnError})
		_, err := svc.GetRepo(t.Context(), "owner", "repo")
		conformance.RequireProviderError(t, err)
	})

	t.Run("list issues success", func(t *testing.T) {
		svc := factory(t, Config{
			Issues: []*ability.ForgeIssue{
				{ID: 100, Index: 1, Title: "First issue", State: "open"},
				{ID: 200, Index: 2, Title: "Second issue", State: "closed"},
			},
		})
		result, err := svc.ListIssues(t.Context(), "owner", &ListIssuesQuery{Page: ability.PageRequest{Limit: 20}})
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Items)
		require.NotNil(t, result.Page)
		assert.Len(t, result.Items, 2)
	})

	t.Run("list issues empty", func(t *testing.T) {
		svc := factory(t, Config{})
		result, err := svc.ListIssues(t.Context(), "owner", &ListIssuesQuery{})
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Items)
		assert.Empty(t, result.Items)
	})

	t.Run("list issues nil query", func(t *testing.T) {
		svc := factory(t, Config{
			Issues: []*ability.ForgeIssue{{ID: 100, Index: 1, Title: "test", State: "open"}},
		})
		result, err := svc.ListIssues(t.Context(), "owner", nil)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("list issues timeout", func(t *testing.T) {
		svc := factory(t, Config{})
		_, err := svc.ListIssues(conformance.CanceledContext(), "", nil)
		conformance.RequireTimeoutError(t, err)
	})

	t.Run("list issues empty owner", func(t *testing.T) {
		svc := factory(t, Config{})
		_, err := svc.ListIssues(t.Context(), "", nil)
		conformance.RequireInvalidArgError(t, err)
	})

	t.Run("list issues provider error", func(t *testing.T) {
		svc := factory(t, Config{IssuesErr: assert.AnError})
		_, err := svc.ListIssues(t.Context(), "owner", &ListIssuesQuery{})
		conformance.RequireProviderError(t, err)
	})

	t.Run("get issue success", func(t *testing.T) {
		svc := factory(t, Config{
			Issues: []*ability.ForgeIssue{{ID: 100, Index: 1, Title: "Test", State: "open"}},
		})
		issue, err := svc.GetIssue(t.Context(), "owner", "repo", 1)
		require.NoError(t, err)
		require.NotNil(t, issue)
		assert.Equal(t, int64(1), issue.Index)
	})

	t.Run("get issue timeout", func(t *testing.T) {
		svc := factory(t, Config{})
		_, err := svc.GetIssue(conformance.CanceledContext(), "", "", 0)
		conformance.RequireTimeoutError(t, err)
	})

	t.Run("get issue empty owner", func(t *testing.T) {
		svc := factory(t, Config{})
		_, err := svc.GetIssue(t.Context(), "", "repo", 1)
		conformance.RequireInvalidArgError(t, err)
	})

	t.Run("get issue provider error", func(t *testing.T) {
		svc := factory(t, Config{IssuesErr: assert.AnError})
		_, err := svc.GetIssue(t.Context(), "owner", "repo", 1)
		conformance.RequireProviderError(t, err)
	})

	t.Run("get commit diff success", func(t *testing.T) {
		svc := factory(t, Config{
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
		svc := factory(t, Config{})
		_, err := svc.GetCommitDiff(conformance.CanceledContext(), "", "", "")
		conformance.RequireTimeoutError(t, err)
	})

	t.Run("get commit diff empty owner", func(t *testing.T) {
		svc := factory(t, Config{})
		_, err := svc.GetCommitDiff(t.Context(), "", "repo", "abc123")
		conformance.RequireInvalidArgError(t, err)
	})

	t.Run("get commit diff empty commit id", func(t *testing.T) {
		svc := factory(t, Config{})
		_, err := svc.GetCommitDiff(t.Context(), "owner", "repo", "")
		conformance.RequireInvalidArgError(t, err)
	})

	t.Run("get commit diff provider error", func(t *testing.T) {
		svc := factory(t, Config{DiffErr: assert.AnError})
		_, err := svc.GetCommitDiff(t.Context(), "owner", "repo", "abc123")
		conformance.RequireProviderError(t, err)
	})

	t.Run("get file content success", func(t *testing.T) {
		svc := factory(t, Config{
			FileContent: []byte("package main"),
		})
		content, err := svc.GetFileContent(t.Context(), "owner", "repo", "abc123", "main.go", 0, 0)
		require.NoError(t, err)
		assert.Equal(t, []byte("package main"), content)
	})

	t.Run("get file content timeout", func(t *testing.T) {
		svc := factory(t, Config{})
		_, err := svc.GetFileContent(conformance.CanceledContext(), "", "", "", "", 0, 0)
		conformance.RequireTimeoutError(t, err)
	})

	t.Run("get file content empty owner", func(t *testing.T) {
		svc := factory(t, Config{})
		_, err := svc.GetFileContent(t.Context(), "", "repo", "abc123", "main.go", 0, 0)
		conformance.RequireInvalidArgError(t, err)
	})

	t.Run("get file content empty file path", func(t *testing.T) {
		svc := factory(t, Config{})
		_, err := svc.GetFileContent(t.Context(), "owner", "repo", "abc123", "", 0, 0)
		conformance.RequireInvalidArgError(t, err)
	})

	t.Run("get file content provider error", func(t *testing.T) {
		svc := factory(t, Config{FileContentErr: assert.AnError})
		_, err := svc.GetFileContent(t.Context(), "owner", "repo", "abc123", "main.go", 0, 0)
		conformance.RequireProviderError(t, err)
	})

	t.Run("list notifications success", func(t *testing.T) {
		svc := factory(t, Config{
			Notifications: []*ability.Notification{
				{ID: "n-1", Reason: "mention", Unread: true, Subject: "PR #42"},
				{ID: "n-2", Reason: "assign", Unread: false, Subject: "Issue #7"},
			},
		})
		result, err := svc.ListNotifications(t.Context(), &PageQuery{Page: ability.PageRequest{Limit: 10}})
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Items)
		assert.Len(t, result.Items, 2)
	})

	t.Run("list notifications timeout", func(t *testing.T) {
		svc := factory(t, Config{})
		_, err := svc.ListNotifications(conformance.CanceledContext(), nil)
		conformance.RequireTimeoutError(t, err)
	})

	t.Run("list notifications provider error", func(t *testing.T) {
		svc := factory(t, Config{NotificationsErr: assert.AnError})
		_, err := svc.ListNotifications(t.Context(), &PageQuery{})
		conformance.RequireProviderError(t, err)
	})

	t.Run("list releases success", func(t *testing.T) {
		svc := factory(t, Config{
			Releases: []*ability.Release{
				{ID: 1, TagName: "v1.0.0", Name: "First release"},
				{ID: 2, TagName: "v2.0.0", Name: "Second release", Prerelease: true},
			},
		})
		result, err := svc.ListReleases(t.Context(), "owner", "repo", &PageQuery{Page: ability.PageRequest{Limit: 10}})
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Items)
		assert.Len(t, result.Items, 2)
	})

	t.Run("list releases empty owner", func(t *testing.T) {
		svc := factory(t, Config{})
		_, err := svc.ListReleases(t.Context(), "", "repo", &PageQuery{})
		conformance.RequireInvalidArgError(t, err)
	})

	t.Run("list releases provider error", func(t *testing.T) {
		svc := factory(t, Config{ReleasesErr: assert.AnError})
		_, err := svc.ListReleases(t.Context(), "owner", "repo", &PageQuery{})
		conformance.RequireProviderError(t, err)
	})
}
