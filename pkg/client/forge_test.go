package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForgeGetUser(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantLogin  string
		wantErr    bool
		errContain string
	}{
		{
			name: "user found",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"id":1,"username":"testuser","email":"test@example.com"}}`))
			},
			wantLogin: "testuser",
		},
		{
			name: "empty user",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"id":0,"username":""}}`))
			},
			wantLogin: "",
		},
		{
			name: "api error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"status":"failed","message":"forge unavailable"}`))
			},
			wantErr:    true,
			errContain: "forge unavailable",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()
			c := NewClient(server.URL, "token")
			result, err := c.Forge.GetUser(context.Background())
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantLogin, result.UserName)
		})
	}
}

func TestForgeGetRepo(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		owner      string
		repo       string
		wantName   string
		wantErr    bool
		errContain string
	}{
		{
			name: "repo found",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"id":1,"name":"myrepo","full_name":"owner/myrepo","private":false,"html_url":"https://example.com"}}`))
			},
			owner:    "owner",
			repo:     "myrepo",
			wantName: "myrepo",
		},
		{
			name:       "missing owner",
			owner:      "",
			repo:       "myrepo",
			wantErr:    true,
			errContain: "owner and repo are required",
		},
		{
			name:       "missing repo",
			owner:      "owner",
			repo:       "",
			wantErr:    true,
			errContain: "owner and repo are required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()
			c := NewClient(server.URL, "token")
			result, err := c.Forge.GetRepo(context.Background(), tt.owner, tt.repo)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantName, result.Name)
		})
	}
}

func TestForgeListIssues(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		owner      string
		query      *ListIssuesQuery
		wantCount  int
		wantErr    bool
		errContain string
	}{
		{
			name: "issues found",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"id":1,"number":1,"title":"bug","state":"open","html_url":"https://example.com/1"},{"id":2,"number":2,"title":"feat","state":"closed","html_url":"https://example.com/2"}]}`))
			},
			owner:     "owner",
			wantCount: 2,
		},
		{
			name: "empty issues",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			},
			owner:     "owner",
			wantCount: 0,
		},
		{
			name:       "missing owner",
			owner:      "",
			wantErr:    true,
			errContain: "owner is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()
			c := NewClient(server.URL, "token")
			result, err := c.Forge.ListIssues(context.Background(), tt.owner, tt.query)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			assert.Len(t, result, tt.wantCount)
		})
	}
}

func TestForgeGetIssue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		owner      string
		repo       string
		index      int64
		wantTitle  string
		wantErr    bool
		errContain string
	}{
		{
			name: "issue found",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"id":1,"number":1,"title":"First Issue","state":"open","html_url":"https://example.com/1"}}`))
			},
			owner:     "owner",
			repo:      "myrepo",
			index:     1,
			wantTitle: "First Issue",
		},
		{
			name:       "missing owner",
			owner:      "",
			repo:       "myrepo",
			index:      1,
			wantErr:    true,
			errContain: "owner and repo are required",
		},
		{
			name:       "invalid index",
			owner:      "owner",
			repo:       "myrepo",
			index:      0,
			wantErr:    true,
			errContain: "index must be positive",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()
			c := NewClient(server.URL, "token")
			result, err := c.Forge.GetIssue(context.Background(), tt.owner, tt.repo, tt.index)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantTitle, result.Title)
		})
	}
}

func TestForgeGetCommitDiff(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		owner      string
		repo       string
		commitID   string
		wantMsg    string
		wantErr    bool
		errContain string
	}{
		{
			name: "diff found",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"commit_id":"abc123","commit_message":"fix: bug","files":["main.go"],"diff_content":"- old\n+ new"}}`))
			},
			owner:    "owner",
			repo:     "myrepo",
			commitID: "abc123",
			wantMsg:  "fix: bug",
		},
		{
			name:       "missing commit_id",
			owner:      "owner",
			repo:       "myrepo",
			commitID:   "",
			wantErr:    true,
			errContain: "commit_id are required",
		},
		{
			name:       "missing owner",
			owner:      "",
			repo:       "myrepo",
			commitID:   "abc123",
			wantErr:    true,
			errContain: "owner, repo and commit_id are required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()
			c := NewClient(server.URL, "token")
			result, err := c.Forge.GetCommitDiff(context.Background(), tt.owner, tt.repo, tt.commitID)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantMsg, result.CommitMessage)
		})
	}
}

func TestForgeGetFileContent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		owner      string
		repo       string
		commitID   string
		filePath   string
		query      *FileContentQuery
		wantOK     bool
		wantErr    bool
		errContain string
	}{
		{
			name: "content found",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":"package main\n\nfunc main() {}"}`))
			},
			owner:    "owner",
			repo:     "myrepo",
			commitID: "abc123",
			filePath: "main.go",
			wantOK:   true,
		},
		{
			name:       "missing file_path",
			owner:      "owner",
			repo:       "myrepo",
			commitID:   "abc123",
			filePath:   "",
			wantErr:    true,
			errContain: "file_path are required",
		},
		{
			name:       "missing owner",
			owner:      "",
			repo:       "myrepo",
			commitID:   "abc123",
			filePath:   "main.go",
			wantErr:    true,
			errContain: "owner, repo, commit_id and file_path are required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()
			c := NewClient(server.URL, "token")
			result, err := c.Forge.GetFileContent(context.Background(), tt.owner, tt.repo, tt.commitID, tt.filePath, tt.query)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			if tt.wantOK {
				assert.NotEmpty(t, result)
			}
		})
	}
}
