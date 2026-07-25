package pages_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/views/pages"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

func TestWorkflowListPageIncludesStatsScripts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		wantContains []string
	}{
		{
			name: "loads chart.js and workflow-stats for overview charts",
			wantContains: []string{
				"/static/vendor/chart.js.min.js",
				"/static/js/workflow-stats.js",
				`hx-get="/service/web/workflows/stats?days=30&amp;groupBy=day"`,
				`data-testid="workflow-list-panel"`,
			},
		},
		{
			name: "does not omit stats scripts when list is empty",
			wantContains: []string{
				"workflow-stats.js",
				"chart.js.min.js",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := pages.WorkflowListPage([]partials.WorkflowListEntry{}).Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			body := buf.String()
			for _, want := range tt.wantContains {
				if !strings.Contains(body, want) {
					t.Fatalf("want %q in body", want)
				}
			}
		})
	}
}

func TestWorkflowRunsPageIncludesStatsScripts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		want string
	}{
		{name: "chart.js", want: "/static/vendor/chart.js.min.js"},
		{name: "workflow-stats.js", want: "/static/js/workflow-stats.js"},
		{name: "stats hx loader", want: "/stats?days=30&amp;groupBy=day"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := pages.WorkflowRunsPage("demo", nil).Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			if !strings.Contains(buf.String(), tt.want) {
				t.Fatalf("want %q in body", tt.want)
			}
		})
	}
}
