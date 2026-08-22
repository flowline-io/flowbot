package pages_test

import (
	"github.com/flowline-io/flowbot/pkg/i18n"
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/views/pages"
)

func TestPipelineRunLivePageCSPSafeExpressions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		bad  string
	}{
		{name: "no JSON global in x-text", bad: "JSON.stringify"},
		{name: "no optional chaining", bad: "?."},
		{name: "no nullish coalescing", bad: "??"},
		{name: "no arrow functions", bad: "=>"},
	}
	var buf bytes.Buffer
	params := pages.PipelineRunLiveParams{
		RunID:        1,
		PipelineName: "demo",
		Trigger:      "event",
		TotalSteps:   1,
		RunStatus:    "running",
		Steps: []pages.StepState{
			{Name: "step-1", Status: "running"},
		},
	}
	if err := pages.PipelineRunLivePage(i18n.DefaultContext(), params).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if strings.Contains(html, tt.bad) {
				t.Fatalf("CSP Alpine cannot parse %q in live dashboard HTML", tt.bad)
			}
		})
	}
	t.Run("uses prettyJSON helper", func(t *testing.T) {
		t.Parallel()
		if !strings.Contains(html, `x-text="prettyJSON(selectedStep.input)"`) {
			t.Fatal("want prettyJSON for step input")
		}
		if !strings.Contains(html, `x-text="prettyJSON(selectedStep.output)"`) {
			t.Fatal("want prettyJSON for step output")
		}
	})
	t.Run("script is cache busted", func(t *testing.T) {
		t.Parallel()
		if !strings.Contains(html, "/static/js/pipeline-run-live.js?v=") {
			t.Fatal("want pipeline-run-live.js?v= cache buster")
		}
	})
}
