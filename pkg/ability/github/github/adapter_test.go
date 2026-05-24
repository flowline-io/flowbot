package github

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/ability"
	githubsvc "github.com/flowline-io/flowbot/pkg/ability/github"
	"github.com/flowline-io/flowbot/pkg/ability/conformance"
	provider "github.com/flowline-io/flowbot/pkg/providers/github"
)

// fakeClient implements the client interface for testing.
type fakeClient struct {
	user             *provider.User
	userErr          error
	repo             *provider.Repository
	userByLogin      *provider.User
	userByLoginErr   error
	repoErr          error
	issues           []*provider.Issue
	issuesErr        error
	diff             *provider.CommitDiff
	diffErr          error
	fileContent      []byte
	fileContentErr   error
	notifications    []*provider.Notification
	notificationsErr error
	releases         []*provider.RepositoryRelease
	releasesErr      error
}

func (f *fakeClient) GetAuthenticatedUser() (*provider.User, error) {
	if f.userErr != nil {
		return nil, f.userErr
	}
	return f.user, nil
}
func (f *fakeClient) GetUser(_ string) (*provider.User, error) {
	if f.userByLoginErr != nil {
		return nil, f.userByLoginErr
	}
	return f.userByLogin, nil
}
func (f *fakeClient) GetRepository(_, _ string) (*provider.Repository, error) {
	if f.repoErr != nil {
		return nil, f.repoErr
	}
	return f.repo, nil
}
func (f *fakeClient) ListIssues(_ string, _, _ int, _ string) ([]*provider.Issue, error) {
	if f.issuesErr != nil {
		return nil, f.issuesErr
	}
	return f.issues, nil
}
func (f *fakeClient) GetDiff(_, _, _ string) (*provider.CommitDiff, error) {
	if f.diffErr != nil {
		return nil, f.diffErr
	}
	return f.diff, nil
}
func (f *fakeClient) GetFileContent(_, _, _, _ string, _, _ int) ([]byte, error) {
	if f.fileContentErr != nil {
		return nil, f.fileContentErr
	}
	return f.fileContent, nil
}
func (f *fakeClient) GetNotifications() ([]*provider.Notification, error) {
	if f.notificationsErr != nil {
		return nil, f.notificationsErr
	}
	return f.notifications, nil
}
func (f *fakeClient) GetReleases(_, _ string, _, _ int) ([]*provider.RepositoryRelease, error) {
	if f.releasesErr != nil {
		return nil, f.releasesErr
	}
	return f.releases, nil
}

func TestAdapter_GetUser(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		client  *fakeClient
		want    *ability.ForgeUser
		wantErr bool
	}{
		{
			name:    "success",
			client:  &fakeClient{user: testUser(1, "testuser", "test@example.com")},
			want:    &ability.ForgeUser{ID: 1, UserName: "testuser", Email: "test@example.com"},
			wantErr: false,
		},
		{
			name:    "provider error",
			client:  &fakeClient{userErr: errors.New("api error")},
			wantErr: true,
		},
		{
			name:    "nil user",
			client:  &fakeClient{user: nil},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			user, err := a.GetUser(context.Background())
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.want == nil {
				assert.Nil(t, user)
			} else {
				require.NotNil(t, user)
				assert.Equal(t, tt.want.ID, user.ID)
				assert.Equal(t, tt.want.UserName, user.UserName)
			}
		})
	}
}

func TestAdapter_GetRepo(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		client  *fakeClient
		owner   string
		repo    string
		want    *ability.ForgeRepo
		wantErr bool
	}{
		{
			name:   "success",
			client: &fakeClient{repo: testRepo(100, "myrepo", "owner/myrepo", "owner")},
			owner:  "owner", repo: "myrepo",
			want:    &ability.ForgeRepo{ID: 100, Name: "myrepo", FullName: "owner/myrepo", Owner: "owner"},
			wantErr: false,
		},
		{
			name:   "empty owner",
			client: &fakeClient{repo: testRepo(100, "r", "o/r", "o")},
			owner:  "", repo: "r",
			wantErr: true,
		},
		{
			name:    "provider error",
			client:  &fakeClient{repoErr: errors.New("api error")},
			owner:   "o", repo: "r",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			repo, err := a.GetRepo(context.Background(), tt.owner, tt.repo)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, repo)
			assert.Equal(t, tt.want.ID, repo.ID)
			assert.Equal(t, tt.want.Name, repo.Name)
		})
	}
}

func TestAdapter_ListIssues(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		client  *fakeClient
		owner   string
		wantLen int
		wantErr bool
	}{
		{
			name: "success with items",
			client: &fakeClient{issues: []*provider.Issue{
				testIssue(1, "First"), testIssue(2, "Second"),
			}},
			owner:   "owner",
			wantLen: 2,
		},
		{
			name:    "empty list",
			client:  &fakeClient{issues: []*provider.Issue{}},
			owner:   "owner",
			wantLen: 0,
		},
		{
			name:    "empty owner",
			client:  &fakeClient{issues: []*provider.Issue{testIssue(1, "First")}},
			owner:   "",
			wantErr: true,
		},
		{
			name:    "provider error",
			client:  &fakeClient{issuesErr: errors.New("api error")},
			owner:   "owner",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			result, err := a.ListIssues(context.Background(), tt.owner, &githubsvc.ListIssuesQuery{Page: ability.PageRequest{Limit: 20}})
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Len(t, result.Items, tt.wantLen)
		})
	}
}

func TestAdapter_GetIssue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		client  *fakeClient
		owner   string
		repo    string
		index   int64
		wantIdx int64
		wantErr bool
	}{
		{
			name: "success",
			client: &fakeClient{issues: []*provider.Issue{
				testIssue(1, "Test Issue"),
			}},
			owner: "owner", repo: "repo", index: 1,
			wantIdx: 1,
		},
		{
			name: "not found",
			client: &fakeClient{issues: []*provider.Issue{
				testIssue(1, "Test Issue"),
			}},
			owner: "owner", repo: "repo", index: 99,
			wantErr: true,
		},
		{
			name: "empty owner",
			client: &fakeClient{issues: []*provider.Issue{testIssue(1, "Test")}},
			owner: "", repo: "repo", index: 1,
			wantErr: true,
		},
		{
			name:    "provider error",
			client:  &fakeClient{issuesErr: errors.New("api error")},
			owner:   "owner", repo: "repo", index: 1,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			issue, err := a.GetIssue(context.Background(), tt.owner, tt.repo, tt.index)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, issue)
			assert.Equal(t, tt.wantIdx, issue.Index)
		})
	}
}

func TestAdapter_GetCommitDiff(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		client  *fakeClient
		owner   string
		commit  string
		wantErr bool
	}{
		{
			name: "success",
			client: &fakeClient{diff: &provider.CommitDiff{
				CommitID: "abc123", CommitMessage: "test", Files: []string{"main.go"}, DiffContent: "diff",
			}},
			owner:  "owner",
			commit: "abc123",
		},
		{
			name:    "empty owner",
			client:  &fakeClient{diff: &provider.CommitDiff{CommitID: "abc"}},
			owner:   "",
			commit:  "abc",
			wantErr: true,
		},
		{
			name:    "provider error",
			client:  &fakeClient{diffErr: errors.New("api error")},
			owner:   "owner",
			commit:  "abc",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			diff, err := a.GetCommitDiff(context.Background(), tt.owner, "repo", tt.commit)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, diff)
			assert.Equal(t, "abc123", diff.CommitID)
		})
	}
}

func TestAdapter_GetFileContent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		client  *fakeClient
		owner   string
		wantErr bool
	}{
		{
			name:   "success",
			client: &fakeClient{fileContent: []byte("content")},
			owner:  "owner",
		},
		{
			name:    "empty owner",
			client:  &fakeClient{fileContent: []byte("content")},
			owner:   "",
			wantErr: true,
		},
		{
			name:    "provider error",
			client:  &fakeClient{fileContentErr: errors.New("api error")},
			owner:   "owner",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			content, err := a.GetFileContent(context.Background(), tt.owner, "repo", "abc", "main.go", 0, 0)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, []byte("content"), content)
		})
	}
}

func TestAdapter_ListNotifications(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		client  *fakeClient
		wantLen int
		wantErr bool
	}{
		{
			name: "success with items",
			client: &fakeClient{notifications: []*provider.Notification{
				testNotification("n-1", "mention", true),
				testNotification("n-2", "assign", false),
			}},
			wantLen: 2,
		},
		{
			name:    "empty list",
			client:  &fakeClient{notifications: []*provider.Notification{}},
			wantLen: 0,
		},
		{
			name:    "provider error",
			client:  &fakeClient{notificationsErr: errors.New("api error")},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			result, err := a.ListNotifications(context.Background(), &githubsvc.PageQuery{})
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Len(t, result.Items, tt.wantLen)
		})
	}
}

func TestAdapter_ListReleases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		client  *fakeClient
		owner   string
		wantLen int
		wantErr bool
	}{
		{
			name: "success with items",
			client: &fakeClient{releases: []*provider.RepositoryRelease{
				testRelease(1, "v1.0.0"),
				testRelease(2, "v2.0.0"),
			}},
			owner:   "owner",
			wantLen: 2,
		},
		{
			name:    "empty list",
			client:  &fakeClient{releases: []*provider.RepositoryRelease{}},
			owner:   "owner",
			wantLen: 0,
		},
		{
			name:    "empty owner",
			client:  &fakeClient{releases: []*provider.RepositoryRelease{testRelease(1, "v1")}},
			owner:   "",
			wantErr: true,
		},
		{
			name:    "provider error",
			client:  &fakeClient{releasesErr: errors.New("api error")},
			owner:   "owner",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			result, err := a.ListReleases(context.Background(), tt.owner, "repo", &githubsvc.PageQuery{})
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Len(t, result.Items, tt.wantLen)
		})
	}
}

func TestFakeClientSatisfiesInterface(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		assert func(t *testing.T)
	}{
		{
			name: "fake client satisfies client interface",
			assert: func(_ *testing.T) {
				var _ client = (*fakeClient)(nil)
			},
		},
		{
			name: "nil check does not panic",
			assert: func(t *testing.T) {
				var c client
				assert.Nil(t, c)
			},
		},
		{
			name: "NewWithClient accepts fakeClient",
			assert: func(t *testing.T) {
				a := NewWithClient(&fakeClient{})
				require.NotNil(t, a)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.assert(t)
		})
	}
}

func TestDecodeTestCursor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		cursor  string
		wantErr bool
	}{
		{
			name:   "valid cursor decodes",
			cursor: func() string { c, _ := ability.EncodeCursor(conformance.CursorSecret, ability.CursorPayload{Capability: "github", Backend: "github", Strategy: "offset", ProviderCursor: "2", Limit: 20}); return c }(),
		},
		{
			name:    "invalid base64",
			cursor:  "!!!invalid!!!",
			wantErr: true,
		},
		{
			name:    "empty cursor",
			cursor:  "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a, ok := NewWithClient(&fakeClient{}).(*Adapter)
			if !ok {
				t.Fatal("unexpected type")
			}
			a.SetCursorSecret(conformance.CursorSecret)
			_, err := decodeTestCursor(t, a, tt.cursor)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func decodeTestCursor(t *testing.T, adapter *Adapter, cursor string) (ability.CursorPayload, error) {
	t.Helper()
	return ability.DecodeCursor(adapter.cursorSecret, cursor, adapter.now())
}

func testUser(id int64, login, email string) *provider.User {
	return &provider.User{
		ID:       &id,
		Login:    &login,
		Email:    &email,
		AvatarURL: strPtr("https://example.com/avatar.png"),
	}
}

func testRepo(id int64, name, fullName, owner string) *provider.Repository {
	return &provider.Repository{
		ID:       &id,
		Name:     &name,
		FullName: &fullName,
		Owner:    &provider.User{Login: &owner},
	}
}

func testIssue(number int, title string) *provider.Issue {
	return &provider.Issue{
		ID:         int64Ptr(int64(number * 100)),
		Number:     &number,
		Title:      &title,
		Body:       strPtr("body"),
		State:      strPtr("open"),
		HTMLURL:    strPtr("https://github.com/owner/repo/issues/" + string(rune('0'+number))),
		User:       &provider.User{Login: strPtr("author")},
		Repository: &provider.Repository{Name: strPtr("repo")},
	}
}

func testNotification(id, reason string, unread bool) *provider.Notification {
	return &provider.Notification{
		ID:     &id,
		Reason: &reason,
		Unread: &unread,
		Subject: &provider.Subject{
			Title: strPtr("subject"),
		},
		Repository: &provider.Repository{
			FullName: strPtr("owner/repo"),
		},
	}
}

func testRelease(id int64, tagName string) *provider.RepositoryRelease {
	return &provider.RepositoryRelease{
		ID:      &id,
		TagName: &tagName,
		Name:    &tagName,
		Body:    strPtr("release body"),
	}
}

func strPtr(s string) *string  { return &s }
func intPtr(i int) *int       { return &i }
func int64Ptr(i int64) *int64 { return &i }
func boolPtr(b bool) *bool    { return &b }
