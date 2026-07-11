package chatagent_test

import (
	"slices"
	"testing"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestApplyToolScope(t *testing.T) {
	t.Parallel()
	chatagent.LockAppConfigForTest(t)

	all := chatagent.ActiveToolNames()
	tests := []struct {
		name             string
		in               chatagent.ToolScopeInput
		wantSchedule     bool
		wantReadOnlyOnly bool
	}{
		{
			name: "plan mode",
			in: chatagent.ToolScopeInput{
				Mode: chatagent.ModePlan, Kind: chatagent.RunKindInteractive, AllActive: all,
			},
			wantReadOnlyOnly: true,
		},
		{
			name: "normal excludes schedule",
			in: chatagent.ToolScopeInput{
				Mode: chatagent.ModeNormal, Kind: chatagent.RunKindInteractive, UserText: "read a file", AllActive: all,
			},
			wantSchedule: false,
		},
		{
			name: "schedule intent includes schedule",
			in: chatagent.ToolScopeInput{
				Mode: chatagent.ModeNormal, Kind: chatagent.RunKindInteractive, UserText: "please schedule a daily reminder", AllActive: all,
			},
			wantSchedule: true,
		},
		{
			name: "scheduled run includes schedule",
			in: chatagent.ToolScopeInput{
				Mode: chatagent.ModeNormal, Kind: chatagent.RunKindScheduled, UserText: "run", AllActive: all,
			},
			wantSchedule: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := chatagent.ApplyToolScope(tt.in)
			hasList := slices.Contains(got, "list_scheduled_tasks")
			if tt.wantReadOnlyOnly {
				assert.Equal(t, chatagent.ReadOnlyToolNames(), got)
				return
			}
			assert.Equal(t, tt.wantSchedule, hasList)
		})
	}
}

func TestApplyToolScopeKeepsAbilityTools(t *testing.T) {
	t.Parallel()
	chatagent.LockAppConfigForTest(t)

	prev := config.App.ChatAgent.AbilityTools
	t.Cleanup(func() { config.App.ChatAgent.AbilityTools = prev })
	config.App.ChatAgent.AbilityTools = []config.AbilityToolConfig{{
		Name: "list_bookmarks", Capability: "bookmark", Operation: "list", Readonly: true,
	}}

	tests := []struct {
		name string
		mode string
		text string
	}{
		{name: "normal keeps ability", mode: chatagent.ModeNormal, text: "read a file"},
		{name: "plan keeps ability", mode: chatagent.ModePlan, text: "research"},
		{name: "schedule intent keeps ability", mode: chatagent.ModeNormal, text: "please schedule a daily reminder"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := chatagent.ApplyToolScope(chatagent.ToolScopeInput{
				Mode:      tt.mode,
				Kind:      chatagent.RunKindInteractive,
				UserText:  tt.text,
				AllActive: chatagent.ActiveToolNames(),
			})
			assert.True(t, slices.Contains(got, "list_bookmarks"))
		})
	}
}
