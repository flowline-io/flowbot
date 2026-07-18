package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/notify/manifest"
)

func TestExtractTemplateFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tmpl string
		want []string
	}{
		{
			name: "extracts dotted fields with pipes",
			tmpl: `**URL:** {{ .url | default "N/A" }}
{{ if .title }}**Title:** {{ .title }}{{ end }}`,
			want: []string{"title", "url"},
		},
		{
			name: "deduplicates fields",
			tmpl: `{{ .body }}{{ .body | html }}{{ .name }}`,
			want: []string{"body", "name"},
		},
		{
			name: "empty template",
			tmpl: "",
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, ExtractTemplateFields(tt.tmpl))
		})
	}
}

func TestSamplePayloadJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		tmpl    manifest.Template
		wantSub []string
	}{
		{
			name: "includes fields from default and overrides",
			tmpl: manifest.Template{
				ID:              "bookmark.created",
				DefaultTemplate: `{{ .url }} {{ .title }}`,
				Overrides: []manifest.Override{
					{Channel: "telegram", Template: `{{ .extra }}`},
				},
			},
			wantSub: []string{`"url"`, `"title"`, `"extra"`, `"summary"`, "bookmark.created"},
		},
		{
			name: "always includes summary",
			tmpl: manifest.Template{
				ID:              "agent.status",
				DefaultTemplate: `{{ .status }}`,
			},
			wantSub: []string{`"status"`, `"summary"`, "agent.status"},
		},
		{
			name: "empty body still has summary",
			tmpl: manifest.Template{
				ID:              "plain",
				DefaultTemplate: "static text",
			},
			wantSub: []string{`"summary"`, "plain"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := SamplePayloadJSON(tt.tmpl)
			require.NoError(t, err)
			for _, sub := range tt.wantSub {
				assert.Contains(t, got, sub)
			}
		})
	}
}
