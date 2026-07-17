package coding_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/coding"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func textFromResult(t *testing.T, result msg.ToolResultMessage) string {
	t.Helper()
	part, ok := result.Parts[0].(msg.TextPart)
	require.True(t, ok)
	return part.Text
}

func TestReadFileTool_Execute(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "hello.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("hello world"), 0o644))
	multiPath := filepath.Join(root, "lines.txt")
	require.NoError(t, os.WriteFile(multiPath, []byte("line1\nline2\nline3\nline4\n"), 0o644))

	tests := []struct {
		name      string
		path      string
		args      map[string]any
		wantText  string
		wantError bool
	}{
		{name: "reads existing file", path: "hello.txt", args: map[string]any{"path": "hello.txt"}, wantText: "hello world"},
		{name: "strips file uri prefix", path: "hello.txt", args: map[string]any{"path": "file://hello.txt"}, wantText: "hello world"},
		{name: "missing file", path: "missing.txt", args: map[string]any{"path": "missing.txt"}, wantError: true},
		{name: "traversal blocked", path: "../secret.txt", args: map[string]any{"path": "../secret.txt"}, wantError: true},
		{name: "offset and limit", path: "lines.txt", args: map[string]any{"path": "lines.txt", "offset": 2, "limit": 2}, wantText: "line2\nline3"},
		{name: "offset beyond file", path: "lines.txt", args: map[string]any{"path": "lines.txt", "offset": 10}, wantText: ""},
	}

	tool := coding.ReadFileTool{Workspace: coding.Workspace{Root: root}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := tool.Execute(context.Background(), "id-1", tt.args, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.wantError, result.IsError)
			if !tt.wantError {
				assert.Equal(t, tt.wantText, textFromResult(t, result))
			}
		})
	}
}

func TestWriteFileTool_Execute(t *testing.T) {
	root := t.TempDir()
	tool := coding.WriteFileTool{Workspace: coding.Workspace{Root: root}}

	tests := []struct {
		name      string
		path      string
		content   string
		wantError bool
	}{
		{name: "writes file", path: "nested/out.txt", content: "data"},
		{name: "strips file uri prefix", path: "file://nested/prefixed.txt", content: "data"},
		{name: "traversal blocked", path: "../bad.txt", content: "x", wantError: true},
		{name: "empty path", path: "", content: "x", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := tool.Execute(context.Background(), "id", map[string]any{
				"path": tt.path, "content": tt.content,
			}, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.wantError, result.IsError)
			if !tt.wantError {
				checkPath := tt.path
				if after, ok := strings.CutPrefix(checkPath, "file://"); ok {
					checkPath = after
				}
				_, statErr := os.Stat(filepath.Join(root, checkPath))
				assert.NoError(t, statErr)
			}
		})
	}
}

func TestRunTerminalTool_Execute(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("platform-specific shell assertions")
	}
	root := t.TempDir()
	tool := coding.RunTerminalTool{Workspace: coding.Workspace{Root: root}}

	tests := []struct {
		name      string
		command   string
		wantError bool
	}{
		{name: "echo command", command: "echo hello", wantError: false},
		{name: "empty command", command: "   ", wantError: true},
		{name: "invalid command", command: "exit 9", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := tool.Execute(context.Background(), "id", map[string]any{"command": tt.command}, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.wantError, result.IsError)
		})
	}
}

func TestWebSearchTool_Execute(t *testing.T) {
	htmlServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`
<a class="result__a" href="https://example.com/go">Go language</a>
<a class="result__snippet">An open-source programming language.</a>
`))
	}))
	defer htmlServer.Close()

	tests := []struct {
		name      string
		query     string
		wantError bool
		wantText  string
	}{
		{name: "valid query returns organic hit", query: "golang", wantError: false, wantText: "Go language"},
		{name: "empty query", query: "  ", wantError: true},
		{name: "whitespace trimmed", query: " go ", wantError: false, wantText: "example.com/go"},
	}

	tool := coding.WebSearchTool{HTTPClient: htmlServer.Client(), BaseURL: htmlServer.URL}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(context.Background(), "id", map[string]any{"query": tt.query}, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.wantError, result.IsError)
			if tt.wantText != "" {
				assert.Contains(t, textFromResult(t, result), tt.wantText)
			}
		})
	}
}

func TestWebSearchTool_SearxPreferred(t *testing.T) {
	t.Parallel()
	searx := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "amd 9070gre", r.URL.Query().Get("q"))
		assert.Equal(t, "json", r.URL.Query().Get("format"))
		_, _ = w.Write([]byte(`{"results":[{"title":"RX 9070 GRE","url":"https://shop.example/9070","content":"Price CNY 4599"}]}`))
	}))
	defer searx.Close()

	htmlBlocked := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("duckduckgo html should not be called when searx is configured")
	}))
	defer htmlBlocked.Close()

	tool := coding.WebSearchTool{
		HTTPClient: searx.Client(),
		BaseURL:    htmlBlocked.URL,
		SearxURL:   searx.URL,
	}
	result, err := tool.Execute(context.Background(), "id", map[string]any{"query": "amd 9070gre"}, nil)
	require.NoError(t, err)
	require.False(t, result.IsError)
	text := textFromResult(t, result)
	assert.Contains(t, text, "RX 9070 GRE")
	assert.Contains(t, text, "4599")
}

func TestWebSearchTool_BraveAPI(t *testing.T) {
	t.Parallel()
	brave := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "secret", r.Header.Get("X-Subscription-Token"))
		_, _ = w.Write([]byte(`{"web":{"results":[{"title":"Brave hit","url":"https://brave.example","description":"snippet"}]}}`))
	}))
	defer brave.Close()

	tool := coding.WebSearchTool{
		HTTPClient:   brave.Client(),
		BraveAPIKey:  "secret",
		BraveBaseURL: brave.URL,
	}
	result, err := tool.Execute(context.Background(), "id", map[string]any{"query": "test"}, nil)
	require.NoError(t, err)
	require.False(t, result.IsError)
	assert.Contains(t, textFromResult(t, result), "Brave hit")
}

func TestWebSearchTool_CaptchaReturnsHint(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<form id="challenge-form" action="//duckduckgo.com/anomaly.js"></form>`))
	}))
	defer server.Close()

	tool := coding.WebSearchTool{HTTPClient: server.Client(), BaseURL: server.URL}
	result, err := tool.Execute(context.Background(), "id", map[string]any{"query": "price"}, nil)
	require.NoError(t, err)
	require.True(t, result.IsError)
	assert.Contains(t, textFromResult(t, result), "searx_url")
}

func TestRunCodeTool_Execute(t *testing.T) {
	root := t.TempDir()
	tool := coding.RunCodeTool{Workspace: coding.Workspace{Root: root}}

	tests := []struct {
		name      string
		language  string
		code      string
		wantError bool
	}{
		{name: "unsupported language", language: "rust", code: "fn main(){}", wantError: true},
		{name: "missing code", language: "python", code: "  ", wantError: true},
		{name: "missing language", language: "", code: "print(1)", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := tool.Execute(context.Background(), "id", map[string]any{
				"language": tt.language,
				"code":     tt.code,
			}, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.wantError, result.IsError)
		})
	}
}
