package pages_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/views/pages"
)

func TestPipelineListPageIncludesStatsScripts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		wantContains []string
	}{
		{
			name: "loads chart.js and pipeline-stats for overview charts",
			wantContains: []string{
				"/static/vendor/chart.js.min.js",
				"/static/js/pipeline-stats.js",
				`hx-get="/service/web/pipelines/stats?groupBy=day"`,
				`data-testid="btn-new-pipeline"`,
			},
		},
		{
			name: "keeps create modal and list container",
			wantContains: []string{
				`data-testid="create-modal"`,
				`data-testid="pipeline-list-container"`,
			},
		},
		{
			name: "does not omit stats scripts when list is empty",
			wantContains: []string{
				"pipeline-stats.js",
				"chart.js.min.js",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := pages.PipelineListPage([]*gen.PipelineDefinition{}).Render(context.Background(), &buf)
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

func TestPipelineRunsPageIncludesStatsScripts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		want string
	}{
		{name: "chart.js", want: "/static/vendor/chart.js.min.js"},
		{name: "pipeline-stats.js", want: "/static/js/pipeline-stats.js"},
		{name: "stats hx loader", want: "/stats?groupBy=day"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := pages.PipelineRunsPage("demo", nil).Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			if !strings.Contains(buf.String(), tt.want) {
				t.Fatalf("want %q in body", tt.want)
			}
		})
	}
}
