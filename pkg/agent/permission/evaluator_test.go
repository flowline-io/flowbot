package permission_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/permission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvaluatorRulePriority(t *testing.T) {
	cfg := permission.Config{
		"bash": {
			Patterns: []permission.PatternRule{
				{Pattern: "*", Action: permission.ActionAsk},
				{Pattern: "git *", Action: permission.ActionAllow},
				{Pattern: "git push *", Action: permission.ActionDeny},
			},
		},
	}
	eval := permission.NewEvaluator(cfg)
	req := permission.Request{Tool: permission.ToolRunTerminal, Args: map[string]any{"command": "git push origin main"}}
	got := eval.Evaluate(req, permission.NewSessionState())
	assert.Equal(t, permission.ActionDeny, got.Action)
}

func TestEvaluatorAlwaysGrant(t *testing.T) {
	cfg := permission.Config{"bash": {Default: permission.ActionAsk}}
	eval := permission.NewEvaluator(cfg)
	sess := permission.NewSessionState()
	require.NoError(t, sess.AddGrant("bash", "git status*"))
	req := permission.Request{Tool: permission.ToolRunTerminal, Args: map[string]any{"command": "git status"}}
	got := eval.Evaluate(req, sess)
	assert.Equal(t, permission.ActionAllow, got.Action)
}

func TestEvaluatorDoomLoop(t *testing.T) {
	eval := permission.NewEvaluator(permission.DefaultConfig())
	sess := permission.NewSessionState()
	args := map[string]any{"command": "ls"}
	req := permission.Request{Tool: permission.ToolRunTerminal, Args: args}
	var got permission.Result
	for range 3 {
		got = eval.Evaluate(req, sess)
	}
	assert.True(t, got.DoomLoopTriggered)
	assert.Equal(t, permission.ActionAsk, got.Action)
}

func TestEvaluatorExternalPath(t *testing.T) {
	cfg := permission.Config{
		permission.KeyExternalDirectory: {Default: permission.ActionDeny},
		"bash":                          {Default: permission.ActionAllow},
	}
	eval := permission.NewEvaluator(cfg)
	root := t.TempDir()
	req := permission.Request{
		Tool:          permission.ToolRunTerminal,
		Args:          map[string]any{"command": "cat /outside/file"},
		WorkspaceRoot: root,
	}
	got := eval.Evaluate(req, permission.NewSessionState())
	assert.Equal(t, permission.ActionDeny, got.Action)
	assert.Equal(t, permission.KeyExternalDirectory, got.PermissionKey)
}

func TestEvaluatorEnvFileDeny(t *testing.T) {
	eval := permission.NewEvaluator(permission.DefaultConfig())
	req := permission.Request{Tool: permission.ToolReadFile, Args: map[string]any{"path": "secrets.env"}}
	got := eval.Evaluate(req, permission.NewSessionState())
	assert.Equal(t, permission.ActionDeny, got.Action)
}

func TestEvaluatorChainStricter(t *testing.T) {
	cfg := permission.Config{"bash": {Default: permission.ActionAllow}}
	eval := permission.NewEvaluator(cfg)
	req := permission.Request{Tool: permission.ToolRunTerminal, Args: map[string]any{"command": "git status && rm -rf /"}}
	got := eval.Evaluate(req, permission.NewSessionState())
	assert.Equal(t, permission.ActionAsk, got.Action)
}

func TestEvaluatorDelegateAndSchedule(t *testing.T) {
	eval := permission.NewEvaluator(permission.DefaultConfig())
	tests := []struct {
		name string
		tool string
		args map[string]any
		want permission.Action
		key  string
	}{
		{
			name: "task defaults ask",
			tool: permission.ToolTask,
			args: map[string]any{"subagent_type": "explore"},
			want: permission.ActionAsk,
			key:  permission.KeyDelegate,
		},
		{
			name: "schedule create defaults ask",
			tool: permission.ToolScheduleTask,
			args: map[string]any{"name": "daily report"},
			want: permission.ActionAsk,
			key:  permission.KeySchedule,
		},
		{
			name: "schedule list defaults allow",
			tool: permission.ToolListScheduledTasks,
			args: map[string]any{},
			want: permission.ActionAllow,
			key:  permission.KeyScheduleRead,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := eval.Evaluate(permission.Request{Tool: tt.tool, Args: tt.args}, permission.NewSessionState())
			assert.Equal(t, tt.want, got.Action)
			assert.Equal(t, tt.key, got.PermissionKey)
		})
	}
}
