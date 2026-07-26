package chatagent

import (
	"context"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadSkillToolAllowlist(t *testing.T) {
	tests := []struct {
		name      string
		allowed   []string
		skillName string
		wantErr   string
	}{
		{
			name:      "disallowed skill rejected before store lookup",
			allowed:   []string{"allowed-only"},
			skillName: "blocked",
			wantErr:   "not available to this agent",
		},
		{
			name:      "allowed skill reaches store lookup",
			allowed:   []string{"demo"},
			skillName: "demo",
			wantErr:   "read skill",
		},
		{
			name:      "skill prefix stripped before allowlist check",
			allowed:   []string{"demo"},
			skillName: "skill://demo",
			wantErr:   "read skill",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := NewReadSkillTool(tt.allowed)
			result, err := tool.Execute(context.Background(), "call-1", map[string]any{
				"name": tt.skillName,
			}, nil)
			require.NoError(t, err)
			assert.True(t, result.IsError)
			assert.Contains(t, skillToolResultText(result), tt.wantErr)
		})
	}
}

func TestReadSkillToolNilName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		args    map[string]any
		wantErr string
	}{
		{name: "nil name", args: map[string]any{"name": nil}, wantErr: "skill name is required"},
		{name: "missing name", args: map[string]any{}, wantErr: "skill name is required"},
		{name: "string nil sentinel", args: map[string]any{"name": "<nil>"}, wantErr: "skill name is required"},
		{name: "slash prefix reaches lookup", args: map[string]any{"name": "/demo"}, wantErr: "read skill"},
		{name: "skill key fallback", args: map[string]any{"skill": "/demo"}, wantErr: "read skill"},
		{name: "nil path ignored", args: map[string]any{"name": "demo", "path": nil}, wantErr: "read skill"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tool := NewReadSkillTool(nil)
			result, err := tool.Execute(context.Background(), "call-1", tt.args, nil)
			require.NoError(t, err)
			assert.True(t, result.IsError)
			assert.Contains(t, skillToolResultText(result), tt.wantErr)
			assert.NotContains(t, skillToolResultText(result), `"<nil>"`)
		})
	}
}

func TestNormalizeReadSkillName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args map[string]any
		want string
	}{
		{name: "plain name", args: map[string]any{"name": "gitea"}, want: "gitea"},
		{name: "slash chip prefix", args: map[string]any{"name": "/gitea"}, want: "gitea"},
		{name: "skill location prefix", args: map[string]any{"name": "skill://gitea"}, want: "gitea"},
		{name: "nil name rejected", args: map[string]any{"name": nil}, want: ""},
		{name: "missing name", args: map[string]any{}, want: ""},
		{name: "string nil sentinel", args: map[string]any{"name": "<nil>"}, want: ""},
		{name: "null sentinel", args: map[string]any{"name": "null"}, want: ""},
		{name: "skill key fallback", args: map[string]any{"skill": "/fireflyiii"}, want: "fireflyiii"},
		{name: "nil args", args: nil, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, normalizeReadSkillName(tt.args))
		})
	}
}

func skillToolResultText(result msg.ToolResultMessage) string {
	var out strings.Builder
	for _, part := range result.Parts {
		if tp, ok := part.(msg.TextPart); ok {
			_, _ = out.WriteString(tp.Text)
		}
	}
	return out.String()
}
