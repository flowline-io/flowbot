package chatagent

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/dcg"
	"github.com/flowline-io/flowbot/pkg/agent/hooks"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/permission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterHooksDCGBeforePermission(t *testing.T) {
	LockAppConfigForTest(t)

	tests := []struct {
		name        string
		checker     dcg.Checker
		tool        string
		args        map[string]any
		wantBlock   bool
		wantReason  string
		notReason   string
		skipDCGOnly bool
	}{
		{
			name:       "deny blocks before permission ask",
			checker:    dcg.DenyChecker{Reason: "dcg blocked test"},
			tool:       permission.ToolRunTerminal,
			args:       map[string]any{"command": "ls"},
			wantBlock:  true,
			wantReason: "dcg blocked test",
		},
		{
			name:      "allow reaches permission layer",
			checker:   dcg.AllowAllChecker{},
			tool:      permission.ToolRunTerminal,
			args:      map[string]any{"command": "ls"},
			wantBlock: true,
			notReason: "dcg blocked test",
		},
		{
			name:        "non shell tool skips dcg",
			checker:     dcg.DenyChecker{Reason: "should not run"},
			tool:        permission.ToolReadFile,
			args:        map[string]any{"path": "note.txt"},
			skipDCGOnly: true,
			notReason:   "should not run",
		},
		{
			name:       "run_code deny",
			checker:    dcg.DenyChecker{Reason: "code blocked"},
			tool:       permission.ToolRunCode,
			args:       map[string]any{"language": "python", "code": "print(1)"},
			wantBlock:  true,
			wantReason: "code blocked",
		},
		{
			name:      "checker error fail closed",
			checker:   dcg.ErrorChecker{},
			tool:      permission.ToolRunTerminal,
			args:      map[string]any{"command": "echo ok"},
			wantBlock: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := hooks.NewRegistry()
			RegisterHooks(reg, ChatHookDeps{
				SessionID: "dcg-hook-test",
				DCG:       tt.checker,
			})
			result, err := reg.EmitToolCall(context.Background(), hooks.ToolCallEvent{
				ToolCall: msg.ToolCallPart{Name: tt.tool},
				Args:     tt.args,
			})
			require.NoError(t, err)
			if tt.skipDCGOnly {
				if result != nil && result.Block {
					assert.NotEqual(t, tt.notReason, result.Reason)
				}
				return
			}
			require.NotNil(t, result)
			assert.Equal(t, tt.wantBlock, result.Block)
			if tt.wantReason != "" {
				assert.Equal(t, tt.wantReason, result.Reason)
			}
			if tt.notReason != "" {
				assert.NotEqual(t, tt.notReason, result.Reason)
			}
		})
	}
}
