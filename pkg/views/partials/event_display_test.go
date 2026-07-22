package partials

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventTypeChipClass(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		eventType string
		want      string
	}{
		{name: "failed suffix", eventType: "job.failed", want: "flowbot-chip flowbot-chip-error"},
		{name: "error substring", eventType: "sync.error", want: "flowbot-chip flowbot-chip-error"},
		{name: "created suffix", eventType: "bookmark.created", want: "flowbot-chip flowbot-chip-success"},
		{name: "success suffix", eventType: "run.success", want: "flowbot-chip flowbot-chip-success"},
		{name: "webhook type", eventType: "webhook.push", want: "flowbot-chip flowbot-chip-primary"},
		{name: "default muted", eventType: "note.updated", want: "flowbot-chip flowbot-chip-muted"},
		{name: "empty muted", eventType: "", want: "flowbot-chip flowbot-chip-muted"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, EventTypeChipClass(tt.eventType))
		})
	}
}

func TestEventSourceChipClass(t *testing.T) {
	t.Parallel()
	allowed := map[string]bool{
		"flowbot-chip flowbot-chip-primary": true,
		"flowbot-chip flowbot-chip-muted":   true,
		"flowbot-chip flowbot-chip-warning": true,
	}
	tests := []struct {
		name   string
		source string
	}{
		{name: "empty muted", source: ""},
		{name: "github in palette", source: "github"},
		{name: "karakeep in palette", source: "karakeep"},
		{name: "stable for same source", source: "miniflux"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EventSourceChipClass(tt.source)
			if tt.source == "" {
				assert.Equal(t, "flowbot-chip flowbot-chip-muted", got)
				return
			}
			assert.True(t, allowed[got], "got %q", got)
			assert.Equal(t, got, EventSourceChipClass(tt.source))
		})
	}
}

func TestEventRunStatusChipClass(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		status string
		want   string
	}{
		{name: "success", status: "2", want: "flowbot-chip flowbot-chip-success"},
		{name: "failed", status: "4", want: "flowbot-chip flowbot-chip-error"},
		{name: "cancelled", status: "3", want: "flowbot-chip flowbot-chip-warning"},
		{name: "running", status: "1", want: "flowbot-chip flowbot-chip-muted"},
		{name: "unknown", status: "99", want: "flowbot-chip flowbot-chip-muted"},
		{name: "empty", status: "", want: "flowbot-chip flowbot-chip-muted"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, EventRunStatusChipClass(tt.status))
		})
	}
}

func TestEventRunStatusText(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		status string
		want   string
	}{
		{name: "success", status: "2", want: "Success"},
		{name: "failed", status: "4", want: "Failed"},
		{name: "cancelled", status: "3", want: "Cancelled"},
		{name: "running", status: "1", want: "Running"},
		{name: "started default", status: "0", want: "Started"},
		{name: "empty started", status: "", want: "Started"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, EventRunStatusText(tt.status))
		})
	}
}

func TestSimilarEventsURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		source    string
		eventType string
		want      string
	}{
		{
			name:      "source and type",
			source:    "github",
			eventType: "webhook.push",
			want:      "/service/web/events?source=github&type=webhook.push",
		},
		{
			name:      "source only",
			source:    "karakeep",
			eventType: "",
			want:      "/service/web/events?source=karakeep",
		},
		{
			name:      "type only",
			source:    "",
			eventType: "bookmark.created",
			want:      "/service/web/events?type=bookmark.created",
		},
		{
			name:      "escapes special chars",
			source:    "a b",
			eventType: "x&y",
			want:      "/service/web/events?source=a+b&type=x%26y",
		},
		{
			name:      "both empty",
			source:    "",
			eventType: "",
			want:      "/service/web/events",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, SimilarEventsURL(tt.source, tt.eventType))
		})
	}
}

func TestPipelineRunLiveURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		pipe string
		id   int64
		want string
	}{
		{name: "basic", pipe: "sync", id: 42, want: "/service/web/pipelines/sync/runs/42/live"},
		{name: "path segment escape", pipe: "a/b", id: 1, want: "/service/web/pipelines/a%2Fb/runs/1/live"},
		{name: "zero id still builds", pipe: "p", id: 0, want: "/service/web/pipelines/p/runs/0/live"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, PipelineRunLiveURL(tt.pipe, tt.id))
		})
	}
}

func TestHighlightJSON(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		contain []string
		absent  []string
	}{
		{
			name:  "object with key string number bool null",
			input: "{\n  \"url\": \"https://x\",\n  \"n\": 1,\n  \"ok\": true,\n  \"v\": null\n}",
			contain: []string{
				`class="flowbot-json-key"`,
				`class="flowbot-json-string"`,
				`class="flowbot-json-number"`,
				`class="flowbot-json-bool"`,
				`class="flowbot-json-null"`,
			},
		},
		{
			name:    "escapes html in strings",
			input:   `{"x":"<script>"}`,
			contain: []string{`&lt;script&gt;`},
			absent:  []string{`<script>`},
		},
		{
			name:    "empty object",
			input:   `{}`,
			contain: []string{`{}`},
		},
		{
			name:    "invalid falls back escaped",
			input:   `{not json`,
			contain: []string{`{not json`},
			absent:  []string{`class="flowbot-json-`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HighlightJSON(tt.input)
			for _, sub := range tt.contain {
				assert.Contains(t, got, sub)
			}
			for _, sub := range tt.absent {
				assert.NotContains(t, got, sub)
			}
		})
	}
}

func TestPrettyJSON(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "indents compact object", input: `{"a":1}`, want: "{\n  \"a\": 1\n}"},
		{name: "empty stays", input: "", want: "{}"},
		{name: "invalid returns original", input: `{bad`, want: `{bad`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PrettyJSON(tt.input)
			require.Equal(t, tt.want, got)
			if strings.Contains(tt.input, `"a"`) {
				assert.Contains(t, HighlightJSON(got), `flowbot-json-key`)
			}
		})
	}
}

func TestWebhookMethodChipClass(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		method string
		want   string
	}{
		{name: "post primary", method: "POST", want: "flowbot-chip flowbot-chip-primary"},
		{name: "delete error", method: "DELETE", want: "flowbot-chip flowbot-chip-error"},
		{name: "get muted", method: "GET", want: "flowbot-chip flowbot-chip-muted"},
		{name: "empty muted", method: "", want: "flowbot-chip flowbot-chip-muted"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, WebhookMethodChipClass(tt.method))
		})
	}
}
