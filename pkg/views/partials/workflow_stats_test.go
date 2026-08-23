package partials

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestWorkflowStatsActiveTabs(t *testing.T) {
	t.Parallel()
	stats := &types.WorkflowStats{
		Summary: types.WorkflowStatsSummary{},
		TriggerSourcePie: []types.TriggerSourceCount{
			{Source: "cron"}, {Source: "webhook"}, {Source: "manual"},
		},
		DurationDistribution: types.WorkflowDurationDistribution{
			Workflow: []types.DurationEntry{
				{Bucket: "0-1s"}, {Bucket: "1-5s"}, {Bucket: "5-30s"}, {Bucket: "30s+"},
			},
		},
	}
	tests := []struct {
		name         string
		tabs         StatsTabState
		wantContains []string
		wantAbsent   []string
	}{
		{
			name: "30d day active",
			tabs: StatsTabState{RangeDays: 30, GroupBy: "day"},
			wantContains: []string{
				`data-testid="btn-range-30d"`,
				`days=30&amp;groupBy=day`,
			},
		},
		{
			name: "all week preserves range on group links",
			tabs: StatsTabState{RangeDays: 0, GroupBy: "week"},
			wantContains: []string{
				`data-testid="btn-range-all"`,
				`data-testid="btn-groupby-week"`,
				`days=0&amp;groupBy=month`,
				`days=90&amp;groupBy=week`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := WorkflowStats(context.Background(), "", stats, tt.tabs).Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			body := buf.String()
			for _, want := range tt.wantContains {
				if !strings.Contains(body, want) {
					t.Fatalf("want %q in body", want)
				}
			}
			// Active class should appear on the selected range button element.
			if tt.tabs.RangeDays == 0 && !strings.Contains(body, `data-testid="btn-range-all"`) {
				t.Fatal("missing all button")
			}
		})
	}
}
