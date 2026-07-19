package partials_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/pipelinedefinition"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

func TestPageHeader(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		title    string
		subtitle string
		want     []string
		absent   []string
	}{
		{
			name:   "title only",
			title:  "Configs",
			want:   []string{`data-testid="page-header"`, "Configs", "font-semibold tracking-tight"},
			absent: []string{"font-bold", "card-title"},
		},
		{
			name:     "with subtitle",
			title:    "Home",
			subtitle: "Operational overview",
			want:     []string{"Home", "Operational overview", "text-base-content/60"},
			absent:   []string{"card-title"},
		},
		{
			name:   "tokens page title",
			title:  "Tokens",
			want:   []string{"Tokens", "text-2xl"},
			absent: []string{"card-title"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			if err := partials.PageHeader(tt.title, tt.subtitle).Render(context.Background(), &buf); err != nil {
				t.Fatalf("render: %v", err)
			}
			body := buf.String()
			for _, w := range tt.want {
				if !strings.Contains(body, w) {
					t.Fatalf("want %q in %s", w, body)
				}
			}
			for _, a := range tt.absent {
				if strings.Contains(body, a) {
					t.Fatalf("did not want %q in %s", a, body)
				}
			}
		})
	}
}

func TestOpsConsoleSurfacesAvoidLegacyCardShell(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		html string
	}{
		{
			name: "config table",
			html: renderTempl(t, partials.ConfigTable(nil)),
		},
		{
			name: "token table",
			html: renderTempl(t, partials.TokenTable(nil)),
		},
		{
			name: "agent skill table",
			html: renderTempl(t, partials.AgentSkillTable(nil)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if strings.Contains(tt.html, "card bg-base-100 shadow-sm") {
				t.Fatal("legacy card bg-base-100 shadow-sm shell still present")
			}
			if !strings.Contains(tt.html, "flowbot-surface") {
				t.Fatal("want flowbot-surface")
			}
		})
	}
}

func TestPipelineListUsesChipsNotBadges(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		entries []partials.PipelineListEntry
		want    string
		absent  string
	}{
		{
			name:    "empty avoids badge classes",
			entries: nil,
			want:    "pipeline-empty",
			absent:  "badge badge-success",
		},
		{
			name: "published uses chip success",
			entries: []partials.PipelineListEntry{
				{
					Definition: &gen.PipelineDefinition{
						Name:   "demo",
						Status: pipelinedefinition.StatusPublished,
					},
					Enabled: true,
				},
			},
			want:   "flowbot-chip-success",
			absent: "badge-success",
		},
		{
			name: "draft uses chip muted",
			entries: []partials.PipelineListEntry{
				{
					Definition: &gen.PipelineDefinition{
						Name:   "draft-one",
						Status: pipelinedefinition.StatusDraft,
					},
					Enabled: false,
				},
			},
			want:   "flowbot-chip-muted",
			absent: "badge-ghost",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			html := renderTempl(t, partials.PipelineListTable(tt.entries))
			if !strings.Contains(html, tt.want) {
				t.Fatalf("want %q in %s", tt.want, html)
			}
			if strings.Contains(html, tt.absent) {
				t.Fatalf("did not want %q", tt.absent)
			}
		})
	}
}

func renderTempl(t *testing.T, c templ.Component) string {
	t.Helper()
	var buf bytes.Buffer
	if err := c.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}
