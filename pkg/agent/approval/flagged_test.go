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
			name: "inline python3 -c",
			req: permission.Request{
				Tool:          permission.ToolRunTerminal,
				Args:          map[string]any{"command": `python3 -c "print(1)"`},
				WorkspaceRoot: root,
			},
			wantFlagged: true,
		},
		{
			name: "inline bash -c",
			req: permission.Request{
				Tool:          permission.ToolRunTerminal,
				Args:          map[string]any{"command": `bash -c "ls"`},
				WorkspaceRoot: root,
			},
			wantFlagged: true,
		},
		{
			name: "inline pwsh -Command",
			req: permission.Request{
				Tool:          permission.ToolRunTerminal,
				Args:          map[string]any{"command": `pwsh -Command Get-Date`},
				WorkspaceRoot: root,
			},
			wantFlagged: true,
		},
		{
			name: "inline cmd /c",
			req: permission.Request{
				Tool:          permission.ToolRunTerminal,
				Args:          map[string]any{"command": `cmd /c dir`},
				WorkspaceRoot: root,
			},
			wantFlagged: true,
		},
		{
			name: "git commit -c not inline script",
			req: permission.Request{
				Tool:          permission.ToolRunTerminal,
				Args:          map[string]any{"command": "git commit -c HEAD"},
				WorkspaceRoot: root,
			},
			wantFlagged: false,
		},
		{
			name: "echo --command not inline script",
			req: permission.Request{
				Tool:          permission.ToolRunTerminal,
				Args:          map[string]any{"command": "echo --command foo"},
				WorkspaceRoot: root,
			},
			wantFlagged: false,
		},
		{
			name: "rm recursive destructive",
			req: permission.Request{
				Tool:          permission.ToolRunTerminal,
				Args:          map[string]any{"command": "rm -rf /tmp/x"},
				WorkspaceRoot: root,
			},
			wantFlagged: true,
		},
		{
			name: "rm plain file not destructive",
			req: permission.Request{
				Tool:          permission.ToolRunTerminal,
				Args:          map[string]any{"command": "rm README.md"},
				WorkspaceRoot: root,
			},
			wantFlagged: false,
		},
		{
			name: "dd of=/dev flagged",
			req: permission.Request{
				Tool:          permission.ToolRunTerminal,
				Args:          map[string]any{"command": "dd of=/dev/sda if=/dev/zero"},
				WorkspaceRoot: root,
			},
			wantFlagged: true,
		},
		{
			name: "echo format disk not destructive",
			req: permission.Request{
				Tool:          permission.ToolRunTerminal,
				Args:          map[string]any{"command": "echo format disk"},
				WorkspaceRoot: root,
			},
			wantFlagged: false,
		},
		{
			name: "format drive destructive",
			req: permission.Request{
				Tool:          permission.ToolRunTerminal,
				Args:          map[string]any{"command": "format C:"},
				WorkspaceRoot: root,
			},
			wantFlagged: true,
		},
		{
			name: "Remove-Item recurse destructive",
			req: permission.Request{
				Tool:          permission.ToolRunTerminal,
				Args:          map[string]any{"command": "Remove-Item -Recurse -Force x"},
				WorkspaceRoot: root,
			},
			wantFlagged: true,
		},
		{
			name: "git reset --hard destructive",
			req: permission.Request{
				Tool:          permission.ToolRunTerminal,
				Args:          map[string]any{"command": "git reset --hard HEAD"},
				WorkspaceRoot: root,
			},
			wantFlagged: true,
		},
		{
			name: "chmod symbolic privilege",
			req: permission.Request{
				Tool:          permission.ToolRunTerminal,
				Args:          map[string]any{"command": "chmod +x bin"},
				WorkspaceRoot: root,
			},
			wantFlagged: true,
		},
		{
			name: "chmod remove mode privilege",
			req: permission.Request{
				Tool:          permission.ToolRunTerminal,
				Args:          map[string]any{"command": "chmod u-w file"},
				WorkspaceRoot: root,
			},
			wantFlagged: true,
		},
		{
			name: "chmod assign mode privilege",
			req: permission.Request{
				Tool:          permission.ToolRunTerminal,
				Args:          map[string]any{"command": "chmod a=rwx dir"},
				WorkspaceRoot: root,
			},
			wantFlagged: true,
		},
		{
			name: "pkexec privilege",
			req: permission.Request{
				Tool:          permission.ToolRunTerminal,
				Args:          map[string]any{"command": "pkexec ls"},
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
			name: "force push plus refspec",
			req: permission.Request{
				Tool:          permission.ToolRunTerminal,
				Args:          map[string]any{"command": "git push origin +main"},
				WorkspaceRoot: root,
			},
			wantFlagged: true,
		},
		{
			name: "echo --force-with-lease not force push",
			req: permission.Request{
				Tool:          permission.ToolRunTerminal,
				Args:          map[string]any{"command": "echo --force-with-lease"},
				WorkspaceRoot: root,
			},
			wantFlagged: false,
		},
		{
			name: "nft network sys",
			req: permission.Request{
				Tool:          permission.ToolRunTerminal,
				Args:          map[string]any{"command": "nft add rule inet filter input drop"},
				WorkspaceRoot: root,
			},
			wantFlagged: true,
		},
		{
			name: "ip route network sys",
			req: permission.Request{
				Tool:          permission.ToolRunTerminal,
				Args:          map[string]any{"command": "ip route add default via 1.1.1.1"},
				WorkspaceRoot: root,
			},
			wantFlagged: true,
		},
		{
			name: "curl network cli",
			req: permission.Request{
				Tool:          permission.ToolRunTerminal,
				Args:          map[string]any{"command": "curl https://example.com"},
				WorkspaceRoot: root,
			},
			wantFlagged: true,
		},
		{
			name: "cat .env sensitive command",
			req: permission.Request{
				Tool:          permission.ToolRunTerminal,
				Args:          map[string]any{"command": "cat .env"},
				WorkspaceRoot: root,
			},
			wantFlagged: true,
		},
		{
			name: "npm publish package risk",
			req: permission.Request{
				Tool:          permission.ToolRunTerminal,
				Args:          map[string]any{"command": "npm publish"},
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
			name: "aws credentials write",
			req: permission.Request{
				Tool:          permission.ToolWriteFile,
				Args:          map[string]any{"path": ".aws/credentials"},
				WorkspaceRoot: root,
			},
			wantFlagged: true,
		},
		{
			name: "mysecrets.txt not sensitive path",
			req: permission.Request{
				Tool:          permission.ToolWriteFile,
				Args:          map[string]any{"path": "mysecrets.txt"},
				WorkspaceRoot: root,
			},
			wantFlagged: false,
		},
		{
			name: "run_code empty flagged",
			req: permission.Request{
				Tool:          permission.ToolRunCode,
				Args:          map[string]any{"language": "python", "code": ""},
				WorkspaceRoot: root,
			},
			wantFlagged: true,
		},
		{
			name: "run_code safe print not flagged",
			req: permission.Request{
				Tool:          permission.ToolRunCode,
				Args:          map[string]any{"language": "python", "code": "print(1)"},
				WorkspaceRoot: root,
			},
			wantFlagged: false,
		},
		{
			name: "run_code dangerous os.remove",
			req: permission.Request{
				Tool:          permission.ToolRunCode,
				Args:          map[string]any{"language": "python", "code": "import os; os.remove('x')"},
				WorkspaceRoot: root,
			},
			wantFlagged: true,
		},
		{
			name: "run_code dangerous eval",
			req: permission.Request{
				Tool:          permission.ToolRunCode,
				Args:          map[string]any{"language": "python", "code": "eval('1+1')"},
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
