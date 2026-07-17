package chatagent_test

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsReadOnlyTool(t *testing.T) {
	tests := []struct {
		name string
		tool string
		want bool
	}{
		{name: "read_file allowed", tool: "read_file", want: true},
		{name: "list_dir allowed", tool: "list_dir", want: true},
		{name: "glob_files allowed", tool: "glob_files", want: true},
		{name: "grep_files allowed", tool: "grep_files", want: true},
		{name: "web_search allowed", tool: "web_search", want: true},
		{name: "web_fetch allowed", tool: "web_fetch", want: true},
		{name: "read_skill allowed", tool: "read_skill", want: true},
		{name: "list_scheduled_tasks allowed", tool: "list_scheduled_tasks", want: true},
		{name: "update_memory not unconditionally read-only", tool: "update_memory", want: false},
		{name: "task blocked in plan mode", tool: "task", want: false},
		{name: "write_file blocked", tool: "write_file", want: false},
		{name: "apply_patch blocked", tool: "apply_patch", want: false},
		{name: "run_terminal blocked", tool: "run_terminal", want: false},
		{name: "run_code blocked", tool: "run_code", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, chatagent.IsReadOnlyTool(tt.tool))
		})
	}
}

func TestReadOnlyToolNames(t *testing.T) {
	tests := []struct {
		name string
		want []string
	}{
		{
			name: "contains read-only set",
			want: []string{
				"list_dir", "glob_files", "grep_files", "read_file",
				"web_search", "web_fetch", "read_skill", "list_scheduled_tasks", "update_memory",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, chatagent.ReadOnlyToolNames())
		})
	}
}

func TestValidSessionMode(t *testing.T) {
	tests := []struct {
		name string
		mode string
		want bool
	}{
		{name: "normal valid", mode: chatagent.ModeNormal, want: true},
		{name: "plan valid", mode: chatagent.ModePlan, want: true},
		{name: "unknown invalid", mode: "execute", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, chatagent.ValidSessionMode(tt.mode))
		})
	}
}

func TestSetSessionModeRejectsInvalidMode(t *testing.T) {
	tests := []struct {
		name string
		mode string
	}{
		{name: "empty mode", mode: ""},
		{name: "unknown mode", mode: "debug"},
		{name: "typo mode", mode: "plans"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := chatagent.SetSessionMode(context.Background(), "sess-1", tt.mode)
			require.Error(t, err)
		})
	}
}

func TestLoadSessionModeDefaults(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
		want      string
	}{
		{name: "empty session id", sessionID: "", want: chatagent.ModeNormal},
		{name: "missing session", sessionID: "missing", want: chatagent.ModeNormal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origDB := store.Database
			store.Database = nil
			t.Cleanup(func() { store.Database = origDB })
			assert.Equal(t, tt.want, chatagent.LoadSessionMode(context.Background(), tt.sessionID))
		})
	}
}
