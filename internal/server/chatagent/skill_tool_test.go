package chatagent_test

import (
	"context"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
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
			tool := chatagent.NewReadSkillTool(tt.allowed)
			result, err := tool.Execute(context.Background(), "call-1", map[string]any{
				"name": tt.skillName,
			}, nil)
			require.NoError(t, err)
			assert.True(t, result.IsError)
			assert.Contains(t, skillToolResultText(result), tt.wantErr)
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
