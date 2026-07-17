package coding_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/coding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebFetchTool_Execute(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		if r.URL.Path == "/big" {
			_, _ = w.Write([]byte(strings.Repeat("x", coding.MaxFetchBytes+10)))
			return
		}
		if r.URL.Path == "/err" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte("hello fetch"))
	}))
	defer server.Close()

	tool := coding.WebFetchTool{HTTPClient: server.Client(), MaxOutput: 2048, AllowLoopback: true}
	tests := []struct {
		name      string
		url       string
		wantError bool
		contains  string
	}{
		{name: "fetches body", url: server.URL + "/ok", contains: "hello fetch"},
		{name: "empty url", url: "  ", wantError: true},
		{name: "blocks localhost", url: "http://127.0.0.1/secret", wantError: true},
		{name: "rejects file scheme", url: "file:///etc/passwd", wantError: true},
		{name: "non 2xx", url: server.URL + "/err", wantError: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fetchTool := tool
			if tt.name == "blocks localhost" {
				fetchTool.AllowLoopback = false
			}
			result, err := fetchTool.Execute(context.Background(), "id", map[string]any{"url": tt.url}, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.wantError, result.IsError)
			if !tt.wantError {
				assert.Contains(t, textFromResult(t, result), tt.contains)
			}
		})
	}
}

func TestWriteFileTool_RejectsOversized(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	tool := coding.WriteFileTool{Workspace: coding.Workspace{Root: root}}
	result, err := tool.Execute(context.Background(), "id", map[string]any{
		"path":    "big.txt",
		"content": strings.Repeat("a", coding.MaxWriteFileBytes+1),
	}, nil)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestWriteFileTool_RejectsMissingContent(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	tool := coding.WriteFileTool{Workspace: coding.Workspace{Root: root}}
	tests := []struct {
		name string
		args map[string]any
	}{
		{name: "nil content", args: map[string]any{"path": "a.txt", "content": nil}},
		{name: "missing content key", args: map[string]any{"path": "a.txt"}},
		{name: "empty path", args: map[string]any{"path": "", "content": "x"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := tool.Execute(context.Background(), "id", tt.args, nil)
			require.NoError(t, err)
			assert.True(t, result.IsError)
		})
	}
}

func TestRedirectChecker_BlocksLoopback(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{name: "blocks 127.0.0.1", url: "http://127.0.0.1/secret", want: true},
		{name: "blocks localhost", url: "http://localhost/secret", want: true},
		{name: "allows example", url: "https://example.com/doc", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			u, err := url.Parse(tt.url)
			require.NoError(t, err)
			check := coding.RedirectChecker(false, nil)
			err = check(&http.Request{URL: u}, []*http.Request{{}})
			if tt.want {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestRunCodeTool_RejectsOversized(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	tool := coding.RunCodeTool{Workspace: coding.Workspace{Root: root}}
	result, err := tool.Execute(context.Background(), "id", map[string]any{
		"language": "python",
		"code":     strings.Repeat("x", coding.MaxRunCodeBytes+1),
	}, nil)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestWebSearchTool_RejectsOversizedQuery(t *testing.T) {
	t.Parallel()
	tool := coding.WebSearchTool{}
	result, err := tool.Execute(context.Background(), "id", map[string]any{
		"query": strings.Repeat("q", coding.MaxWebSearchQueryBytes+1),
	}, nil)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}
