package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGithubGetUser(t *testing.T) {
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
				_, _ = w.Write([]byte(`{"status":"ok","data":{"id":1,"username":"ghuser","email":"gh@example.com"}}`))
			},
			wantLogin: "ghuser",
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
				_, _ = w.Write([]byte(`{"status":"failed","message":"github unavailable"}`))
			},
			wantErr:    true,
			errContain: "github unavailable",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()
			c := NewClient(server.URL, "token")
			result, err := c.Github.GetUser(context.Background())
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

func TestGithubGetUserByLogin(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		login      string
		wantLogin  string
		wantErr    bool
		errContain string
	}{
		{
			name: "user found by login",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"id":42,"username":"someone","email":"someone@example.com"}}`))
			},
			login:     "someone",
			wantLogin: "someone",
		},
		{
			name:       "empty login",
			login:      "",
			wantErr:    true,
			errContain: "login is required",
		},
		{
			name: "user not found",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"status":"failed","retcode":"10009","message":"user not found"}`))
			},
			login:      "nobody",
			wantErr:    true,
			errContain: "user not found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()
			c := NewClient(server.URL, "token")
			result, err := c.Github.GetUserByLogin(context.Background(), tt.login)
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

func TestGithubGetRepo(t *testing.T) {
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
				_, _ = w.Write([]byte(`{"status":"ok","data":{"id":1,"name":"ghrepo","full_name":"owner/ghrepo","private":false,"html_url":"https://github.com/owner/ghrepo"}}`))
			},
			owner:    "owner",
			repo:     "ghrepo",
			wantName: "ghrepo",
		},
		{
			name:       "missing owner",
			owner:      "",
			repo:       "ghrepo",
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
			result, err := c.Github.GetRepo(context.Background(), tt.owner, tt.repo)
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

func TestGithubListIssues(t *testing.T) {
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
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"id":1,"number":1,"title":"bug","state":"open","html_url":"https://github.com/1"},{"id":2,"number":2,"title":"feat","state":"closed","html_url":"https://github.com/2"}]}`))
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
			result, err := c.Github.ListIssues(context.Background(), tt.owner, tt.query)
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

func TestGithubGetIssue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		owner      string
		repo       string
		number     int64
		wantTitle  string
		wantErr    bool
		errContain string
	}{
		{
			name: "issue found",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"id":1,"number":42,"title":"PR Title","state":"open","html_url":"https://github.com/1"}}`))
			},
			owner:     "owner",
			repo:      "ghrepo",
			number:    42,
			wantTitle: "PR Title",
		},
		{
			name:       "missing owner",
			owner:      "",
			repo:       "ghrepo",
			number:     1,
			wantErr:    true,
			errContain: "owner and repo are required",
		},
		{
			name:       "invalid number",
			owner:      "owner",
			repo:       "ghrepo",
			number:     0,
			wantErr:    true,
			errContain: "number must be positive",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()
			c := NewClient(server.URL, "token")
			result, err := c.Github.GetIssue(context.Background(), tt.owner, tt.repo, tt.number)
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

func TestGithubGetCommitDiff(t *testing.T) {
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
				_, _ = w.Write([]byte(`{"status":"ok","data":{"commit_id":"def456","commit_message":"feat: add endpoint","files":["api.go"],"diff_content":"+ new line"}}`))
			},
			owner:    "owner",
			repo:     "ghrepo",
			commitID: "def456",
			wantMsg:  "feat: add endpoint",
		},
		{
			name:       "missing commit_id",
			owner:      "owner",
			repo:       "ghrepo",
			commitID:   "",
			wantErr:    true,
			errContain: "commit_id are required",
		},
		{
			name:       "missing owner",
			owner:      "",
			repo:       "ghrepo",
			commitID:   "def456",
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
			result, err := c.Github.GetCommitDiff(context.Background(), tt.owner, tt.repo, tt.commitID)
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

func TestGithubGetFileContent(t *testing.T) {
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
				_, _ = w.Write([]byte(`{"status":"ok","data":"package main\n"}`))
			},
			owner:    "owner",
			repo:     "ghrepo",
			commitID: "def456",
			filePath: "main.go",
			wantOK:   true,
		},
		{
			name:       "missing file_path",
			owner:      "owner",
			repo:       "ghrepo",
			commitID:   "def456",
			filePath:   "",
			wantErr:    true,
			errContain: "file_path are required",
		},
		{
			name:       "missing owner",
			owner:      "",
			repo:       "ghrepo",
			commitID:   "def456",
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
			result, err := c.Github.GetFileContent(context.Background(), tt.owner, tt.repo, tt.commitID, tt.filePath, tt.query)
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

func TestGithubListNotifications(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		query      *ListNotificationsQuery
		wantCount  int
		wantErr    bool
		errContain string
	}{
		{
			name: "notifications found",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"id":"n1","reason":"mention","unread":true,"subject":"PR #42"},{"id":"n2","reason":"assign","unread":false,"subject":"Issue #7"}]}`))
			},
			wantCount: 2,
		},
		{
			name: "empty notifications",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":[]}`))
			},
			wantCount: 0,
		},
		{
			name: "api error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"status":"failed","message":"github notifications unavailable"}`))
			},
			wantErr:    true,
			errContain: "github notifications unavailable",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(tt.handler)
			defer server.Close()
			c := NewClient(server.URL, "token")
			result, err := c.Github.ListNotifications(context.Background(), tt.query)
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

func TestGithubListReleases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		owner      string
		repo       string
		query      *ListNotificationsQuery
		wantCount  int
		wantErr    bool
		errContain string
	}{
		{
			name: "releases found",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":[{"id":1,"tag_name":"v1.0.0","name":"First Release","draft":false,"prerelease":false},{"id":2,"tag_name":"v2.0.0","name":"Second Release","draft":true,"prerelease":true}]}`))
			},
			owner:     "owner",
			repo:      "ghrepo",
			wantCount: 2,
		},
		{
			name:       "missing owner",
			owner:      "",
			repo:       "ghrepo",
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
			result, err := c.Github.ListReleases(context.Background(), tt.owner, tt.repo, tt.query)
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

func TestGithubGetUserByLogin_URLPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		login    string
		wantPath string
	}{
		{"simple login", "user", "/service/github/user/user"},
		{"login with dots", "user.name", "/service/github/user/user.name"},
		{"login with hyphens", "user-name", "/service/github/user/user-name"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tt.wantPath, r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"id":1,"username":"` + tt.login + `"}}`))
			}))
			defer server.Close()
			c := NewClient(server.URL, "token")
			_, err := c.Github.GetUserByLogin(context.Background(), tt.login)
			require.NoError(t, err)
		})
	}
}
