package pages_test

import (
	"github.com/flowline-io/flowbot/pkg/i18n"
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
	if err := pages.PipelineEditorPage(i18n.DefaultContext(), "demo").Render(context.Background(), &buf); err != nil {
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
	t.Run("int list param input is rendered", func(t *testing.T) {
		t.Parallel()
		if !strings.Contains(html, `x-if="isParamTypeIntList(p)"`) {
			t.Fatal("want []int64 param input branch")
		}
		if !strings.Contains(html, "setStepParamIntList") {
			t.Fatal("want setStepParamIntList wiring")
		}
	})
	t.Run("title click starts rename", func(t *testing.T) {
		t.Parallel()
		if !strings.Contains(html, `data-testid="pipeline-title"`) {
			t.Fatal("want clickable pipeline title")
		}
		if !strings.Contains(html, `@click="startRename"`) {
			t.Fatal("want title click to start rename")
		}
		if strings.Contains(html, `data-testid="btn-rename-pipeline"`) {
			t.Fatal("rename button should not be present")
		}
		if !strings.Contains(html, `data-testid="input-rename-pipeline"`) {
			t.Fatal("want inline rename input")
		}
	})
}

func TestPipelineEditorWebhookURLCopyMarkup(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	if err := pages.PipelineEditorPage(i18n.DefaultContext(), "demo").Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()
	tests := []struct {
		name string
		want string
	}{
		{name: "trigger card url", want: `data-testid="webhook-trigger-url"`},
		{name: "trigger card label helper", want: `x-text="webhookTriggerLabel(t)"`},
		{name: "trigger card copy button", want: `data-testid="btn-copy-webhook-url"`},
		{name: "drawer url display", want: `data-testid="webhook-url-display"`},
		{name: "drawer copy button", want: `data-testid="btn-copy-webhook-url-drawer"`},
		{name: "trigger type change helper", want: `@change="onTriggerTypeChange()"`},
		{name: "auth hint on card", want: `data-testid="webhook-auth-hint"`},
		{name: "copy curl example", want: `data-testid="btn-copy-webhook-curl"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if !strings.Contains(html, tt.want) {
				t.Fatalf("want %q in pipeline editor HTML", tt.want)
			}
		})
	}
}
