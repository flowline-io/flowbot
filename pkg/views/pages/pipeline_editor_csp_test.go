package pages_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/views/pages"
)

func TestPipelineEditorPageCSPSafeExpressions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		bad  string
	}{
		{name: "no optional chaining", bad: "?."},
		{name: "no nullish coalescing", bad: "??"},
		{name: "no arrow functions", bad: "=>"},
		{name: "no array spread in x-for", bad: "...Array"},
		{name: "no number input for template params", bad: `type="number"`},
		{name: "no enabledTriggers method call in x-for", bad: "enabledTriggers()"},
	}
	var buf bytes.Buffer
	if err := pages.PipelineEditorPage("demo").Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if strings.Contains(html, tt.bad) {
				t.Fatalf("CSP Alpine cannot parse %q in pipeline editor HTML", tt.bad)
			}
		})
	}
	t.Run("uses enabledTriggers property in x-for", func(t *testing.T) {
		t.Parallel()
		if !strings.Contains(html, `x-for="t in enabledTriggers"`) {
			t.Fatal("want x-for over enabledTriggers getter property")
		}
	})
	t.Run("pipeline editor script is cache busted", func(t *testing.T) {
		t.Parallel()
		if !strings.Contains(html, "/static/js/pipeline-editor.js?v=") {
			t.Fatal("want pipeline-editor.js?v= cache buster")
		}
	})
}
