package gitea

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/ability"
	provider "github.com/flowline-io/flowbot/pkg/providers/gitea"

	giteasdk "code.gitea.io/sdk/gitea"
)

// fakeClient implements the client interface for testing.
type fakeClient struct {
	user           *giteasdk.User
	userErr        error
	repo           *giteasdk.Repository
	repoErr        error
	issues         []*giteasdk.Issue
	issuesErr      error
	diff           *provider.CommitDiff
	diffErr        error
	fileContent    []byte
	fileContentErr error
}

func (f *fakeClient) GetMyUserInfo() (*giteasdk.User, error) {
	if f.userErr != nil {
		return nil, f.userErr
	}
	if f.user == nil {
		f.user = &giteasdk.User{ID: 1, UserName: "testuser"}
	}
	return f.user, nil
}

func (f *fakeClient) GetRepositories(_, _ string) (*giteasdk.Repository, error) {
	if f.repoErr != nil {
		return nil, f.repoErr
	}
	return f.repo, nil
}

func (f *fakeClient) ListIssues(_ string, _, _ int) ([]*giteasdk.Issue, error) {
	if f.issuesErr != nil {
		return nil, f.issuesErr
	}
	return f.issues, nil
}

func (f *fakeClient) GetDiff(_, _ string, commitID string) (*provider.CommitDiff, error) {
	if f.diffErr != nil {
		return nil, f.diffErr
	}
	if f.diff == nil {
		f.diff = &provider.CommitDiff{CommitID: commitID, CommitMessage: "test commit", Files: []string{"main.go"}, DiffContent: "diff content"}
	}
	return f.diff, nil
}

func (f *fakeClient) GetFileContent(_, _, _, _ string, _, _ int) ([]byte, error) {
	if f.fileContentErr != nil {
		return nil, f.fileContentErr
	}
	if f.fileContent == nil {
		f.fileContent = []byte("package main")
	}
	return f.fileContent, nil
}

func TestAdapter_GetUser(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		client  *fakeClient
		wantErr bool
	}{
		{name: "success", client: &fakeClient{user: &giteasdk.User{ID: 1, UserName: "testuser", Email: "t@e.com"}}, wantErr: false},
		{name: "provider error", client: &fakeClient{userErr: errors.New("down")}, wantErr: true},
		{name: "nil user handled", client: &fakeClient{}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			user, err := a.GetUser(t.Context())
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, user)
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
		wantErr bool
	}{
		{name: "success", client: &fakeClient{repo: &giteasdk.Repository{ID: 1, Name: "repo", FullName: "owner/repo", Owner: &giteasdk.User{UserName: "owner"}}}, owner: "owner", repo: "repo", wantErr: false},
		{name: "empty owner", client: &fakeClient{}, owner: "", repo: "repo", wantErr: true},
		{name: "empty repo name", client: &fakeClient{}, owner: "owner", repo: "", wantErr: true},
		{name: "provider error", client: &fakeClient{repoErr: errors.New("gone")}, owner: "owner", repo: "repo", wantErr: true},
		{name: "nil repo returns not found", client: &fakeClient{}, owner: "owner", repo: "repo", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			r, err := a.GetRepo(t.Context(), tt.owner, tt.repo)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, r)
			assert.Equal(t, tt.repo, r.Name)
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
			name:    "success",
			client:  &fakeClient{issues: []*giteasdk.Issue{testIssue(1, "First"), testIssue(2, "Second")}},
			owner:   "owner",
			wantLen: 2,
			wantErr: false,
		},
		{name: "empty owner", client: &fakeClient{}, owner: "", wantErr: true},
		{name: "provider error", client: &fakeClient{issuesErr: errors.New("fail")}, owner: "owner", wantErr: true},
		{name: "empty result", client: &fakeClient{}, owner: "owner", wantLen: 0, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			result, err := a.ListIssues(t.Context(), tt.owner, nil)
			if tt.wantErr {
				assert.Error(t, err)
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
		wantErr bool
	}{
		{
			name:   "success",
			client: &fakeClient{issues: []*giteasdk.Issue{{ID: 100, Index: 1, Title: "Test", State: giteasdk.StateOpen, Poster: &giteasdk.User{UserName: "a"}, Repository: &giteasdk.RepositoryMeta{Name: "repo"}}}},
			owner:  "owner", repo: "repo", index: 1, wantErr: false,
		},
		{name: "empty owner", client: &fakeClient{}, owner: "", repo: "repo", index: 1, wantErr: true},
		{name: "empty repo", client: &fakeClient{}, owner: "owner", repo: "", index: 1, wantErr: true},
		{name: "provider error", client: &fakeClient{issuesErr: errors.New("fail")}, owner: "owner", repo: "repo", index: 1, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			issue, err := a.GetIssue(t.Context(), tt.owner, tt.repo, tt.index)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, issue)
			assert.Equal(t, tt.index, issue.Index)
		})
	}
}

func TestAdapter_GetCommitDiff(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		client   *fakeClient
		owner    string
		repo     string
		commitID string
		wantErr  bool
	}{
		{name: "success", client: &fakeClient{}, owner: "owner", repo: "repo", commitID: "abc123", wantErr: false},
		{name: "empty owner", client: &fakeClient{}, owner: "", repo: "repo", commitID: "abc123", wantErr: true},
		{name: "empty commit id", client: &fakeClient{}, owner: "owner", repo: "repo", commitID: "", wantErr: true},
		{name: "provider error", client: &fakeClient{diffErr: errors.New("fail")}, owner: "owner", repo: "repo", commitID: "abc123", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			diff, err := a.GetCommitDiff(t.Context(), tt.owner, tt.repo, tt.commitID)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, diff)
			assert.Equal(t, tt.commitID, diff.CommitID)
		})
	}
}

func TestAdapter_GetFileContent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		client   *fakeClient
		owner    string
		repo     string
		commitID string
		filePath string
		wantErr  bool
	}{
		{name: "success", client: &fakeClient{fileContent: []byte("hello")}, owner: "owner", repo: "repo", commitID: "abc123", filePath: "main.go", wantErr: false},
		{name: "empty owner", client: &fakeClient{}, owner: "", repo: "repo", commitID: "abc123", filePath: "main.go", wantErr: true},
		{name: "empty file path", client: &fakeClient{}, owner: "owner", repo: "repo", commitID: "abc123", filePath: "", wantErr: true},
		{name: "provider error", client: &fakeClient{fileContentErr: errors.New("fail")}, owner: "owner", repo: "repo", commitID: "abc123", filePath: "main.go", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := NewWithClient(tt.client)
			content, err := a.GetFileContent(t.Context(), tt.owner, tt.repo, tt.commitID, tt.filePath, 0, 0)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, []byte("hello"), content)
		})
	}
}

func decodeTestCursor(t *testing.T, adapter *Adapter, cursor string) ability.CursorPayload {
	t.Helper()
	payload, err := ability.DecodeCursor(adapter.cursorSecret, cursor, adapter.now())
	require.NoError(t, err)
	return payload
}

func testIssue(index int64, title string) *giteasdk.Issue {
	return &giteasdk.Issue{ID: index * 100, Index: index, Title: title, Body: "body", State: giteasdk.StateOpen, Poster: &giteasdk.User{UserName: "author"}}
}
