package gitea

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/providers"
)

func TestGetWebhookSecret(t *testing.T) {
	tests := []struct {
		name    string
		configs json.RawMessage
		want    string
	}{
		{name: "missing config returns empty", configs: json.RawMessage(`{}`), want: ""},
		{name: "reads secret", configs: json.RawMessage(`{"gitea":{"webhook_secret":"hmac"}}`), want: "hmac"},
		{name: "empty secret", configs: json.RawMessage(`{"gitea":{"webhook_secret":""}}`), want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providers.Configs = tt.configs
			assert.Equal(t, tt.want, GetWebhookSecret())
		})
	}
}

func TestGetClient(t *testing.T) {
	tests := []struct {
		name    string
		configs json.RawMessage
		wantErr bool
	}{
		{name: "empty endpoint fails", configs: json.RawMessage(`{}`), wantErr: true},
		{name: "configured endpoint with mock server", configs: json.RawMessage(`{"gitea":{"endpoint":"__MOCK__","token":"tok"}}`), wantErr: false},
		{name: "endpoint without token", configs: json.RawMessage(`{"gitea":{"endpoint":"__MOCK__"}}`), wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configs := tt.configs
			if strings.Contains(string(tt.configs), "__MOCK__") {
				srv := newGiteaTestServer(t, nil)
				defer srv.Close()
				configs = json.RawMessage(strings.ReplaceAll(string(tt.configs), "__MOCK__", srv.URL))
			}
			providers.Configs = configs
			g, err := GetClient()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, g)
		})
	}
}

func newGiteaTestServer(t *testing.T, routes map[string]http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path
		if key == "GET /api/v1/version" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":"1.21.0"}`))
			return
		}
		if handler, ok := routes[key]; ok {
			handler(w, r)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
}

func TestGitea_GetRepositories(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
		wantName   string
	}{
		{name: "returns repository", statusCode: http.StatusOK, body: `{"name":"repo","full_name":"owner/repo"}`, wantName: "repo"},
		{name: "not found", statusCode: http.StatusNotFound, body: `{}`, wantErr: true},
		{name: "forbidden", statusCode: http.StatusForbidden, body: `{}`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := newGiteaTestServer(t, map[string]http.HandlerFunc{
				"GET /api/v1/repos/owner/repo": func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(tt.statusCode)
					_, _ = w.Write([]byte(tt.body))
				},
			})
			defer srv.Close()

			g, err := NewGitea(srv.URL, "token")
			require.NoError(t, err)

			repo, err := g.GetRepositories("owner", "repo")
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantName, repo.Name)
		})
	}
}

func TestGitea_GetMyUserInfo(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
		wantLogin  string
	}{
		{name: "returns user", statusCode: http.StatusOK, body: `{"login":"dev","id":1}`, wantLogin: "dev"},
		{name: "unauthorized", statusCode: http.StatusUnauthorized, body: `{}`, wantErr: true},
		{name: "server error", statusCode: http.StatusInternalServerError, body: `{}`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := newGiteaTestServer(t, map[string]http.HandlerFunc{
				"GET /api/v1/user": func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(tt.statusCode)
					_, _ = w.Write([]byte(tt.body))
				},
			})
			defer srv.Close()

			g, err := NewGitea(srv.URL, "token")
			require.NoError(t, err)

			user, err := g.GetMyUserInfo()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantLogin, user.UserName)
		})
	}
}

func TestGitea_ListIssues(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantLen    int
		wantErr    bool
	}{
		{name: "returns issues", statusCode: http.StatusOK, body: `[{"number":1,"title":"Bug"}]`, wantLen: 1},
		{name: "empty list", statusCode: http.StatusOK, body: `[]`, wantLen: 0},
		{name: "error response", statusCode: http.StatusBadRequest, body: `{}`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := newGiteaTestServer(t, map[string]http.HandlerFunc{
				"GET /api/v1/repos/issues/search": func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(tt.statusCode)
					_, _ = w.Write([]byte(tt.body))
				},
			})
			defer srv.Close()

			g, err := NewGitea(srv.URL, "token")
			require.NoError(t, err)

			issues, err := g.ListIssues("owner", 1, 10)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, issues, tt.wantLen)
		})
	}
}

func TestGitea_GetCommitDiff(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
		wantSubstr string
	}{
		{name: "returns diff", statusCode: http.StatusOK, body: "diff --git a/main.go", wantSubstr: "diff --git"},
		{name: "not found", statusCode: http.StatusNotFound, body: "", wantErr: true},
		{name: "empty diff ok", statusCode: http.StatusOK, body: "", wantSubstr: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := newGiteaTestServer(t, map[string]http.HandlerFunc{
				"GET /api/v1/repos/owner/repo/git/commits/abc123.diff": func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(tt.statusCode)
					_, _ = w.Write([]byte(tt.body))
				},
			})
			defer srv.Close()

			g, err := NewGitea(srv.URL, "token")
			require.NoError(t, err)

			diff, err := g.GetCommitDiff("owner", "repo", "abc123")
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Contains(t, string(diff), tt.wantSubstr)
		})
	}
}

func TestGitea_GetDiff(t *testing.T) {
	t.Parallel()
	srv := newGiteaTestServer(t, map[string]http.HandlerFunc{
		"GET /api/v1/repos/owner/repo/git/commits/abc123": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"sha":"abc123","commit":{"message":"fix bug"},"files":[{"filename":"main.go"}]}`))
		},
		"GET /api/v1/repos/owner/repo/git/commits/abc123.diff": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("diff --git a/main.go b/main.go"))
		},
	})
	defer srv.Close()

	g, err := NewGitea(srv.URL, "token")
	require.NoError(t, err)

	diff, err := g.GetDiff("owner", "repo", "abc123")
	require.NoError(t, err)
	assert.Equal(t, "abc123", diff.CommitID)
	assert.Equal(t, "fix bug", diff.CommitMessage)
	require.Len(t, diff.Files, 1)
	assert.Contains(t, diff.DiffContent, "diff --git")
}

func TestGitea_GetFileContent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		lineStart int
		lineCount int
		wantSub   string
	}{
		{name: "returns slice of lines", lineStart: 2, lineCount: 1, wantSub: "line2"},
		{name: "start at first line", lineStart: 1, lineCount: 1, wantSub: "line1"},
		{name: "wide window", lineStart: 2, lineCount: 5, wantSub: "line2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := newGiteaTestServer(t, map[string]http.HandlerFunc{
				"GET /api/v1/repos/owner/repo/raw/main.go": func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "abc123", r.URL.Query().Get("ref"))
					w.Header().Set("Content-Type", "text/plain")
					_, _ = w.Write([]byte("line1\nline2\nline3"))
				},
			})
			defer srv.Close()

			g, err := NewGitea(srv.URL, "token")
			require.NoError(t, err)

			content, err := g.GetFileContent("owner", "repo", "abc123", "main.go", tt.lineStart, tt.lineCount)
			require.NoError(t, err)
			assert.Contains(t, string(content), tt.wantSub)
		})
	}
}
