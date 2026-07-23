package permission_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/permission"
	"github.com/stretchr/testify/assert"
)

func TestPermissionKeyForMemoryTools(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		tool string
		want string
	}{
		{name: "memory_set", tool: permission.ToolMemorySet, want: permission.KeyMemory},
		{name: "memory_get", tool: permission.ToolMemoryGet, want: permission.KeyMemory},
		{name: "search_session_summaries", tool: permission.ToolSearchSessionSummaries, want: permission.KeyMemory},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, permission.PermissionKeyForTool(tt.tool))
		})
	}
}

func TestExtractMemoryPatterns(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		tool    string
		wantPat string
	}{
		{name: "set is write", tool: permission.ToolMemorySet, wantPat: "write"},
		{name: "delete is write", tool: permission.ToolMemoryDelete, wantPat: "write"},
		{name: "get is read", tool: permission.ToolMemoryGet, wantPat: "read"},
		{name: "list is list", tool: permission.ToolMemoryList, wantPat: "list"},
		{name: "search summaries is read", tool: permission.ToolSearchSessionSummaries, wantPat: "read"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := permission.ExtractInputs(permission.Request{Tool: tt.tool, Args: map[string]any{}})
			assert.Equal(t, permission.KeyMemory, got.PermissionKey)
			assert.Equal(t, tt.wantPat, got.Primary)
		})
	}
}
