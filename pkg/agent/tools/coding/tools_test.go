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

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tools/coding"
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "google", r.URL.Query().Get("engine"))
		assert.Equal(t, "test-key", r.URL.Query().Get("api_key"))
		assert.Equal(t, "json", r.URL.Query().Get("output"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "search_metadata": {"status": "Success"},
  "organic_results": [
    {"title": "Go language", "link": "https://example.com/go", "snippet": "An open-source programming language."}
  ]
}`))
	}))
	defer server.Close()

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

	tool := coding.WebSearchTool{HTTPClient: server.Client(), BaseURL: server.URL, APIKey: "test-key"}
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

func TestWebSearchTool_SerpAPI(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "amd 9070gre", r.URL.Query().Get("q"))
		assert.Equal(t, "secret", r.URL.Query().Get("api_key"))
		_, _ = w.Write([]byte(`{
  "search_metadata": {"status": "Success"},
  "organic_results": [
    {"title": "RX 9070 GRE", "link": "https://shop.example/9070", "snippet": "Price CNY 4599"}
  ]
}`))
	}))
	defer server.Close()

	tool := coding.WebSearchTool{
		HTTPClient: server.Client(),
		BaseURL:    server.URL,
		APIKey:     "secret",
	}
	result, err := tool.Execute(context.Background(), "id", map[string]any{"query": "amd 9070gre"}, nil)
	require.NoError(t, err)
	require.False(t, result.IsError)
	text := textFromResult(t, result)
	assert.Contains(t, text, "RX 9070 GRE")
	assert.Contains(t, text, "4599")
}

func TestWebSearchTool_FallsBackToDuckDuckGoWhenAPIKeyMissing(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "golang", r.URL.Query().Get("q"))
		assert.Empty(t, r.URL.Query().Get("api_key"))
		assert.Equal(t, "text/html", r.Header.Get("Accept"))
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body>
<a class="result__a" href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fgo.dev%2F">The Go Programming Language</a>
<a class="result__snippet">An open-source programming language.</a>
</body></html>`))
	}))
	defer server.Close()

	tool := coding.WebSearchTool{HTTPClient: server.Client(), BaseURL: server.URL}
	result, err := tool.Execute(context.Background(), "id", map[string]any{"query": "golang"}, nil)
	require.NoError(t, err)
	require.False(t, result.IsError)
	text := textFromResult(t, result)
	assert.Contains(t, text, "The Go Programming Language")
	assert.Contains(t, text, "https://go.dev/")
	assert.Contains(t, text, "open-source")
}

func TestWebSearchTool_DuckDuckGoCapsResults(t *testing.T) {
	t.Parallel()
	var page strings.Builder
	_, _ = page.WriteString("<html><body>")
	for range coding.MaxWebSearchResults + 2 {
		_, _ = page.WriteString(`<a class="result__a" href="https://example.com/x">Hit title</a>`)
	}
	_, _ = page.WriteString("</body></html>")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(page.String()))
	}))
	defer server.Close()

	tool := coding.WebSearchTool{HTTPClient: server.Client(), BaseURL: server.URL}
	result, err := tool.Execute(context.Background(), "id", map[string]any{"query": "cap"}, nil)
	require.NoError(t, err)
	require.False(t, result.IsError)
	assert.Equal(t, coding.MaxWebSearchResults, strings.Count(textFromResult(t, result), "Hit title"))
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
