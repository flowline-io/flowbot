package conformance

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
)

// GithubListIssuesQuery wraps pagination and filtering for listing issues.
type GithubListIssuesQuery = capability.GithubListIssuesQuery

// GithubPageQuery wraps pagination for list operations.
type GithubPageQuery = capability.GithubPageQuery

// GithubService is the GitHub capability contract used by conformance tests.
type GithubService interface {
	GetUser(ctx context.Context) (*capability.ForgeUser, error)
	GetUserByLogin(ctx context.Context, login string) (*capability.ForgeUser, error)
	GetRepo(ctx context.Context, owner, repo string) (*capability.ForgeRepo, error)
	ListIssues(ctx context.Context, owner string, q *GithubListIssuesQuery) (*capability.ListResult[capability.ForgeIssue], error)
	GetIssue(ctx context.Context, owner, repo string, number int64) (*capability.ForgeIssue, error)
	GetCommitDiff(ctx context.Context, owner, repo, commitID string) (*capability.ForgeCommitDiff, error)
	GetFileContent(ctx context.Context, owner, repo, commitID, filePath string, lineStart, lineCount int) ([]byte, error)
	ListNotifications(ctx context.Context, q *GithubPageQuery) (*capability.ListResult[capability.Notification], error)
	ListReleases(ctx context.Context, owner, repo string, q *GithubPageQuery) (*capability.ListResult[capability.Release], error)
}

// GithubConfig configures the fake backend for each GitHub conformance subtest.
type GithubConfig struct {
	User             *capability.ForgeUser
	UserErr          error
	UserByLogin      *capability.ForgeUser
	UserByLoginErr   error
	Repo             *capability.ForgeRepo
	RepoErr          error
	Issues           []*capability.ForgeIssue
	IssuesErr        error
	Diff             *capability.ForgeCommitDiff
	DiffErr          error
	FileContent      []byte
	FileContentErr   error
	Notifications    []*capability.Notification
	NotificationsErr error
	Releases         []*capability.Release
	ReleasesErr      error
}

// GithubServiceFactory creates a fresh GitHub Service wired to a fake backend.
type GithubServiceFactory func(t *testing.T, cfg GithubConfig) GithubService

// RunGithubConformance runs the standard GitHub capability conformance suite.
func RunGithubConformance(t *testing.T, factory GithubServiceFactory) {
	t.Run("get user success", func(t *testing.T) {
		svc := factory(t, GithubConfig{
			User: &capability.ForgeUser{ID: 1, UserName: "testuser", Email: "test@example.com"},
		})
		user, err := svc.GetUser(t.Context())
		require.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, int64(1), user.ID)
		assert.Equal(t, "testuser", user.UserName)
	})

	t.Run("get user timeout", func(t *testing.T) {
		svc := factory(t, GithubConfig{})
		_, err := svc.GetUser(CanceledContext())
		RequireTimeoutError(t, err)
	})

	t.Run("get user provider error", func(t *testing.T) {
		svc := factory(t, GithubConfig{UserErr: assert.AnError})
		_, err := svc.GetUser(t.Context())
		RequireProviderError(t, err)
	})

	t.Run("get user by login success", func(t *testing.T) {
		svc := factory(t, GithubConfig{
			UserByLogin: &capability.ForgeUser{ID: 2, UserName: "otheruser", Email: "other@example.com"},
		})
		user, err := svc.GetUserByLogin(t.Context(), "otheruser")
		require.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, int64(2), user.ID)
		assert.Equal(t, "otheruser", user.UserName)
	})

	t.Run("get user by login timeout", func(t *testing.T) {
		svc := factory(t, GithubConfig{})
		_, err := svc.GetUserByLogin(CanceledContext(), "")
		RequireTimeoutError(t, err)
	})

	t.Run("get user by login empty login", func(t *testing.T) {
		svc := factory(t, GithubConfig{})
		_, err := svc.GetUserByLogin(t.Context(), "")
		RequireInvalidArgError(t, err)
	})

	t.Run("get user by login provider error", func(t *testing.T) {
		svc := factory(t, GithubConfig{UserByLoginErr: assert.AnError})
		_, err := svc.GetUserByLogin(t.Context(), "otheruser")
		RequireProviderError(t, err)
	})

	t.Run("get repo success", func(t *testing.T) {
		svc := factory(t, GithubConfig{
			Repo: &capability.ForgeRepo{ID: 1, Name: "repo", FullName: "owner/repo", Owner: "owner"},
		})
		repo, err := svc.GetRepo(t.Context(), "owner", "repo")
		require.NoError(t, err)
		require.NotNil(t, repo)
		assert.Equal(t, "repo", repo.Name)
		assert.Equal(t, "owner/repo", repo.FullName)
	})

	t.Run("get repo timeout", func(t *testing.T) {
		svc := factory(t, GithubConfig{})
		_, err := svc.GetRepo(CanceledContext(), "", "")
		RequireTimeoutError(t, err)
	})

	t.Run("get repo empty owner", func(t *testing.T) {
		svc := factory(t, GithubConfig{})
		_, err := svc.GetRepo(t.Context(), "", "repo")
		RequireInvalidArgError(t, err)
	})

	t.Run("get repo empty name", func(t *testing.T) {
		svc := factory(t, GithubConfig{})
		_, err := svc.GetRepo(t.Context(), "owner", "")
		RequireInvalidArgError(t, err)
	})

	t.Run("get repo provider error", func(t *testing.T) {
		svc := factory(t, GithubConfig{RepoErr: assert.AnError})
		_, err := svc.GetRepo(t.Context(), "owner", "repo")
		RequireProviderError(t, err)
	})

	t.Run("list issues success", func(t *testing.T) {
		svc := factory(t, GithubConfig{
			Issues: []*capability.ForgeIssue{
				{ID: 100, Index: 1, Title: "First issue", State: "open"},
				{ID: 200, Index: 2, Title: "Second issue", State: "closed"},
			},
		})
		result, err := svc.ListIssues(t.Context(), "owner", &capability.GithubListIssuesQuery{Page: capability.PageRequest{Limit: 20}})
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Items)
		require.NotNil(t, result.Page)
		assert.Len(t, result.Items, 2)
	})

	t.Run("list issues empty", func(t *testing.T) {
		svc := factory(t, GithubConfig{})
		result, err := svc.ListIssues(t.Context(), "owner", &capability.GithubListIssuesQuery{})
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Items)
		assert.Empty(t, result.Items)
	})

	t.Run("list issues nil query", func(t *testing.T) {
		svc := factory(t, GithubConfig{
			Issues: []*capability.ForgeIssue{{ID: 100, Index: 1, Title: "test", State: "open"}},
		})
		result, err := svc.ListIssues(t.Context(), "owner", nil)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("list issues timeout", func(t *testing.T) {
		svc := factory(t, GithubConfig{})
		_, err := svc.ListIssues(CanceledContext(), "", nil)
		RequireTimeoutError(t, err)
	})

	t.Run("list issues empty owner", func(t *testing.T) {
		svc := factory(t, GithubConfig{})
		_, err := svc.ListIssues(t.Context(), "", nil)
		RequireInvalidArgError(t, err)
	})

	t.Run("list issues provider error", func(t *testing.T) {
		svc := factory(t, GithubConfig{IssuesErr: assert.AnError})
		_, err := svc.ListIssues(t.Context(), "owner", &capability.GithubListIssuesQuery{})
		RequireProviderError(t, err)
	})

	t.Run("get issue success", func(t *testing.T) {
		svc := factory(t, GithubConfig{
			Issues: []*capability.ForgeIssue{{ID: 100, Index: 1, Title: "Test", State: "open"}},
		})
		issue, err := svc.GetIssue(t.Context(), "owner", "repo", 1)
		require.NoError(t, err)
		require.NotNil(t, issue)
		assert.Equal(t, int64(1), issue.Index)
	})

	t.Run("get issue timeout", func(t *testing.T) {
		svc := factory(t, GithubConfig{})
		_, err := svc.GetIssue(CanceledContext(), "", "", 0)
		RequireTimeoutError(t, err)
	})

	t.Run("get issue empty owner", func(t *testing.T) {
		svc := factory(t, GithubConfig{})
		_, err := svc.GetIssue(t.Context(), "", "repo", 1)
		RequireInvalidArgError(t, err)
	})

	t.Run("get issue provider error", func(t *testing.T) {
		svc := factory(t, GithubConfig{IssuesErr: assert.AnError})
		_, err := svc.GetIssue(t.Context(), "owner", "repo", 1)
		RequireProviderError(t, err)
	})

	t.Run("get commit diff success", func(t *testing.T) {
		svc := factory(t, GithubConfig{
			Diff: &capability.ForgeCommitDiff{
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
		svc := factory(t, GithubConfig{})
		_, err := svc.GetCommitDiff(CanceledContext(), "", "", "")
		RequireTimeoutError(t, err)
	})

	t.Run("get commit diff empty owner", func(t *testing.T) {
		svc := factory(t, GithubConfig{})
		_, err := svc.GetCommitDiff(t.Context(), "", "repo", "abc123")
		RequireInvalidArgError(t, err)
	})

	t.Run("get commit diff empty commit id", func(t *testing.T) {
		svc := factory(t, GithubConfig{})
		_, err := svc.GetCommitDiff(t.Context(), "owner", "repo", "")
		RequireInvalidArgError(t, err)
	})

	t.Run("get commit diff provider error", func(t *testing.T) {
		svc := factory(t, GithubConfig{DiffErr: assert.AnError})
		_, err := svc.GetCommitDiff(t.Context(), "owner", "repo", "abc123")
		RequireProviderError(t, err)
	})

	t.Run("get file content success", func(t *testing.T) {
		svc := factory(t, GithubConfig{
			FileContent: []byte("package main"),
		})
		content, err := svc.GetFileContent(t.Context(), "owner", "repo", "abc123", "main.go", 0, 0)
		require.NoError(t, err)
		assert.Equal(t, []byte("package main"), content)
	})

	t.Run("get file content timeout", func(t *testing.T) {
		svc := factory(t, GithubConfig{})
		_, err := svc.GetFileContent(CanceledContext(), "", "", "", "", 0, 0)
		RequireTimeoutError(t, err)
	})

	t.Run("get file content empty owner", func(t *testing.T) {
		svc := factory(t, GithubConfig{})
		_, err := svc.GetFileContent(t.Context(), "", "repo", "abc123", "main.go", 0, 0)
		RequireInvalidArgError(t, err)
	})

	t.Run("get file content empty file path", func(t *testing.T) {
		svc := factory(t, GithubConfig{})
		_, err := svc.GetFileContent(t.Context(), "owner", "repo", "abc123", "", 0, 0)
		RequireInvalidArgError(t, err)
	})

	t.Run("get file content provider error", func(t *testing.T) {
		svc := factory(t, GithubConfig{FileContentErr: assert.AnError})
		_, err := svc.GetFileContent(t.Context(), "owner", "repo", "abc123", "main.go", 0, 0)
		RequireProviderError(t, err)
	})

	t.Run("list notifications success", func(t *testing.T) {
		svc := factory(t, GithubConfig{
			Notifications: []*capability.Notification{
				{ID: "n-1", Reason: "mention", Unread: true, Subject: "First", RepoName: "owner/repo"},
				{ID: "n-2", Reason: "assign", Unread: false, Subject: "Second", RepoName: "owner/repo2"},
			},
		})
		result, err := svc.ListNotifications(t.Context(), nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Items)
		require.NotNil(t, result.Page)
		assert.Len(t, result.Items, 2)
	})

	t.Run("list notifications empty", func(t *testing.T) {
		svc := factory(t, GithubConfig{})
		result, err := svc.ListNotifications(t.Context(), &capability.GithubPageQuery{})
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Items)
		assert.Empty(t, result.Items)
	})

	t.Run("list notifications nil query", func(t *testing.T) {
		svc := factory(t, GithubConfig{
			Notifications: []*capability.Notification{
				{ID: "n-1", Reason: "mention", Unread: true, Subject: "First", RepoName: "owner/repo"},
			},
		})
		result, err := svc.ListNotifications(t.Context(), nil)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("list notifications timeout", func(t *testing.T) {
		svc := factory(t, GithubConfig{})
		_, err := svc.ListNotifications(CanceledContext(), nil)
		RequireTimeoutError(t, err)
	})

	t.Run("list notifications provider error", func(t *testing.T) {
		svc := factory(t, GithubConfig{NotificationsErr: assert.AnError})
		_, err := svc.ListNotifications(t.Context(), nil)
		RequireProviderError(t, err)
	})

	t.Run("list releases success", func(t *testing.T) {
		svc := factory(t, GithubConfig{
			Releases: []*capability.Release{
				{ID: 1, TagName: "v1.0.0", Name: "First Release"},
				{ID: 2, TagName: "v2.0.0", Name: "Second Release"},
			},
		})
		result, err := svc.ListReleases(t.Context(), "owner", "repo", nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Items)
		require.NotNil(t, result.Page)
		assert.Len(t, result.Items, 2)
	})

	t.Run("list releases empty", func(t *testing.T) {
		svc := factory(t, GithubConfig{})
		result, err := svc.ListReleases(t.Context(), "owner", "repo", &capability.GithubPageQuery{})
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Items)
		assert.Empty(t, result.Items)
	})

	t.Run("list releases nil query", func(t *testing.T) {
		svc := factory(t, GithubConfig{
			Releases: []*capability.Release{{ID: 1, TagName: "v1.0.0", Name: "Release"}},
		})
		result, err := svc.ListReleases(t.Context(), "owner", "repo", nil)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("list releases timeout", func(t *testing.T) {
		svc := factory(t, GithubConfig{})
		_, err := svc.ListReleases(CanceledContext(), "", "", nil)
		RequireTimeoutError(t, err)
	})

	t.Run("list releases empty owner", func(t *testing.T) {
		svc := factory(t, GithubConfig{})
		_, err := svc.ListReleases(t.Context(), "", "repo", nil)
		RequireInvalidArgError(t, err)
	})

	t.Run("list releases empty repo", func(t *testing.T) {
		svc := factory(t, GithubConfig{})
		_, err := svc.ListReleases(t.Context(), "owner", "", nil)
		RequireInvalidArgError(t, err)
	})

	t.Run("list releases provider error", func(t *testing.T) {
		svc := factory(t, GithubConfig{ReleasesErr: assert.AnError})
		_, err := svc.ListReleases(t.Context(), "owner", "repo", nil)
		RequireProviderError(t, err)
	})
}
