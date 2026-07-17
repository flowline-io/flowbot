package chatagent_test

import (
	"slices"
	"testing"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
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
		{
			name: "pipeline run excludes schedule without intent",
			in: chatagent.ToolScopeInput{
				Mode: chatagent.ModeNormal, Kind: chatagent.RunKindPipeline, UserText: "run", AllActive: all,
			},
			wantSchedule: false,
		},
		{
			name: "pipeline run includes schedule on intent",
			in: chatagent.ToolScopeInput{
				Mode: chatagent.ModeNormal, Kind: chatagent.RunKindPipeline, UserText: "please schedule a daily reminder", AllActive: all,
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
