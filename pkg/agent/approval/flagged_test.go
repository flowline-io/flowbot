package approval_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/approval"
	"github.com/flowline-io/flowbot/pkg/agent/permission"
	"github.com/stretchr/testify/assert"
)

func TestEvaluateFlagged(t *testing.T) {
	root := t.TempDir()
	tests := []struct {
		name        string
		req         permission.Request
		wantFlagged bool
	}{
		{
			name: "readonly read",
			req: permission.Request{
				Tool: permission.ToolReadFile,
				Args: map[string]any{"path": "main.go"},
			},
			wantFlagged: false,
		},
		{
			name: "safe git status",
			req: permission.Request{
				Tool:          permission.ToolRunTerminal,
				Args:          map[string]any{"command": "git status"},
				WorkspaceRoot: root,
			},
			wantFlagged: false,
		},
		{
			name: "inline python -c",
			req: permission.Request{
				Tool:          permission.ToolRunTerminal,
				Args:          map[string]any{"command": `python -c "print(1)"`},
				WorkspaceRoot: root,
			},
			wantFlagged: true,
		},
		{
			name: "rm destructive",
			req: permission.Request{
				Tool:          permission.ToolRunTerminal,
				Args:          map[string]any{"command": "rm -rf /tmp/x"},
				WorkspaceRoot: root,
			},
			wantFlagged: true,
		},
		{
			name: "force push",
			req: permission.Request{
				Tool:          permission.ToolRunTerminal,
				Args:          map[string]any{"command": "git push --force origin main"},
				WorkspaceRoot: root,
			},
			wantFlagged: true,
		},
		{
			name: "chain flagged",
			req: permission.Request{
				Tool:          permission.ToolRunTerminal,
				Args:          map[string]any{"command": "git status && ls"},
				WorkspaceRoot: root,
			},
			wantFlagged: true,
		},
		{
			name: "ordinary write",
			req: permission.Request{
				Tool:          permission.ToolWriteFile,
				Args:          map[string]any{"path": "pkg/foo.go"},
				WorkspaceRoot: root,
			},
			wantFlagged: false,
		},
		{
			name: "env write",
			req: permission.Request{
				Tool:          permission.ToolWriteFile,
				Args:          map[string]any{"path": ".env"},
				WorkspaceRoot: root,
			},
			wantFlagged: true,
		},
		{
			name: "web fetch",
			req: permission.Request{
				Tool: permission.ToolWebFetch,
				Args: map[string]any{"url": "https://example.com"},
			},
			wantFlagged: true,
		},
		{
			name: "external path",
			req: permission.Request{
				Tool:          permission.ToolWriteFile,
				Args:          map[string]any{"path": "/etc/passwd"},
				WorkspaceRoot: root,
				ExternalPath:  true,
			},
			wantFlagged: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := approval.EvaluateFlagged(tt.req)
			assert.Equal(t, tt.wantFlagged, got.Flagged, "reason=%s", got.Reason)
		})
	}
}
