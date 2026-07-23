package permission_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/permission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPermissionKeyForKnowledgeTools(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		tool string
		want string
	}{
		{name: "search_knowledge", tool: permission.ToolSearchKnowledge, want: permission.KeyKnowledge},
		{name: "get_knowledge", tool: permission.ToolGetKnowledge, want: permission.KeyKnowledge},
		{name: "read_skill unchanged", tool: permission.ToolReadSkill, want: "skill"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, permission.PermissionKeyForTool(tt.tool))
		})
	}
}

func TestDefaultConfigAllowsKnowledge(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		key  string
		want permission.Action
	}{
		{name: "knowledge defaults to allow", key: permission.KeyKnowledge, want: permission.ActionAllow},
		{name: "skill defaults to allow", key: "skill", want: permission.ActionAllow},
		{name: "websearch defaults to ask", key: "websearch", want: permission.ActionAsk},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := permission.DefaultConfig()
			rs, ok := cfg[tt.key]
			require.True(t, ok, "missing key %q", tt.key)
			assert.Equal(t, tt.want, rs.Default)
		})
	}
}
