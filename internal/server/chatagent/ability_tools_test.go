package chatagent_test

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateAbilityToolConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     config.AbilityToolConfig
		wantErr string
	}{
		{
			name: "readonly ok",
			cfg: config.AbilityToolConfig{
				Name: "memo_list", Capability: "memo", Operation: "list", Readonly: true,
			},
		},
		{
			name: "rejects non-readonly",
			cfg: config.AbilityToolConfig{
				Name: "memo_create", Capability: "memo", Operation: "create", Readonly: false,
			},
			wantErr: "readonly must be true",
		},
		{
			name: "requires capability",
			cfg: config.AbilityToolConfig{
				Name: "x", Operation: "list", Readonly: true,
			},
			wantErr: "capability is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := chatagent.ValidateAbilityToolConfig(tt.cfg)
			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestAbilityToolExecute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		invoke     chatagent.AbilityInvoker
		args       map[string]any
		wantError  bool
		wantSubstr string
		wantLeak   string
	}{
		{
			name: "success returns text",
			invoke: func(_ context.Context, _ hub.CapabilityType, _ string, _ map[string]any) (*ability.InvokeResult, error) {
				return &ability.InvokeResult{Text: "memo ok"}, nil
			},
			args:       map[string]any{"params": map[string]any{"limit": 1}},
			wantSubstr: "memo ok",
		},
		{
			name: "maps not found without provider leak",
			invoke: func(_ context.Context, _ hub.CapabilityType, _ string, _ map[string]any) (*ability.InvokeResult, error) {
				return nil, types.Errorf(types.ErrNotFound, "item missing")
			},
			wantError:  true,
			wantSubstr: "not_found",
			wantLeak:   "secret-token",
		},
		{
			name: "maps provider error safely",
			invoke: func(_ context.Context, _ hub.CapabilityType, _ string, _ map[string]any) (*ability.InvokeResult, error) {
				return nil, types.WrapError(types.ErrProvider, "raw provider dump secret-token", nil)
			},
			wantError:  true,
			wantSubstr: "unavailable",
			wantLeak:   "secret-token",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			toolItem, err := chatagent.NewAbilityTool(config.AbilityToolConfig{
				Name: "memo_list", Description: "list memos", Capability: "memo", Operation: "list", Readonly: true,
			}, tt.invoke)
			require.NoError(t, err)
			result, err := toolItem.Execute(context.Background(), "call-1", tt.args, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.wantError, result.IsError)
			text := abilityToolResultText(result)
			assert.Contains(t, text, tt.wantSubstr)
			if tt.wantLeak != "" {
				assert.NotContains(t, text, tt.wantLeak)
			}
		})
	}
}

func TestRegisterAbilityTools(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		entries []config.AbilityToolConfig
		wantErr string
		wantLen int
	}{
		{
			name: "registers readonly tools",
			entries: []config.AbilityToolConfig{{
				Name: "memo_list", Capability: "memo", Operation: "list", Readonly: true,
			}},
			wantLen: 1,
		},
		{
			name: "fails when readonly missing",
			entries: []config.AbilityToolConfig{{
				Name: "memo_create", Capability: "memo", Operation: "create",
			}},
			wantErr: "readonly must be true",
		},
		{
			name:    "empty entries ok",
			entries: nil,
			wantLen: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reg := tool.NewRegistry()
			names, err := chatagent.RegisterAbilityTools(reg, tt.entries, nil)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Len(t, names, tt.wantLen)
		})
	}
}

func abilityToolResultText(result msg.ToolResultMessage) string {
	if len(result.Parts) == 0 {
		return ""
	}
	tp, ok := result.Parts[0].(msg.TextPart)
	if !ok {
		return ""
	}
	return tp.Text
}
