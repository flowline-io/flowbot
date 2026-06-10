package coding_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
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

	tests := []struct {
		name      string
		path      string
		wantText  string
		wantError bool
	}{
		{name: "reads existing file", path: "hello.txt", wantText: "hello world"},
		{name: "missing file", path: "missing.txt", wantError: true},
		{name: "traversal blocked", path: "../secret.txt", wantError: true},
	}

	tool := coding.ReadFileTool{Workspace: coding.Workspace{Root: root}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := tool.Execute(context.Background(), "id-1", map[string]any{"path": tt.path}, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.wantError, result.IsError)
			if !tt.wantError {
				assert.Contains(t, textFromResult(t, result), tt.wantText)
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
				_, statErr := os.Stat(filepath.Join(root, tt.path))
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Heading":"Go","AbstractText":"Go is a programming language.","RelatedTopics":[{"Text":"Golang docs"}]}`))
	}))
	defer server.Close()

	tests := []struct {
		name      string
		query     string
		wantError bool
	}{
		{name: "valid query", query: "golang", wantError: false},
		{name: "empty query", query: "  ", wantError: true},
		{name: "whitespace trimmed", query: " go ", wantError: false},
	}

	tool := coding.WebSearchTool{HTTPClient: server.Client(), BaseURL: server.URL}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(context.Background(), "id", map[string]any{"query": tt.query}, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.wantError, result.IsError)
		})
	}
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
