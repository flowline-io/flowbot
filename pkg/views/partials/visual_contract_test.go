package partials_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/flowline-io/flowbot/pkg/i18n"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

func TestPageHeader(t *testing.T) {
	t.Parallel()
	ctx := i18n.DefaultContext()
	tests := []struct {
		name        string
		titleKey    string
		subtitleKey string
		want        []string
		absent      []string
	}{
		{
			name:     "title only",
			titleKey: "nav.configs",
			want:     []string{`data-testid="page-header"`, "Configs", "font-semibold tracking-tight"},
			absent:   []string{"font-bold", "card-title"},
		},
		{
			name:        "with subtitle",
			titleKey:    "nav.home",
			subtitleKey: "page.home.subtitle",
			want:        []string{"Home", "Operational overview", "text-base-content/60"},
			absent:      []string{"card-title"},
		},
		{
			name:     "tokens page title",
			titleKey: "nav.tokens",
			want:     []string{"Tokens", "text-2xl"},
			absent:   []string{"card-title"},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			if err := partials.PageHeader(ctx, tt.titleKey, tt.subtitleKey).Render(ctx, &buf); err != nil {
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

func TestPageHeaderI18n(t *testing.T) {
	t.Parallel()
	enCtx := i18n.DefaultContext()
	zhCtx := i18n.WithLocalizer(context.Background(), i18n.LocalizerForCookie(i18n.CookieZH))

	var enBuf bytes.Buffer
	if err := partials.PageHeader(enCtx, "nav.configs", "page.home.subtitle").Render(enCtx, &enBuf); err != nil {
		t.Fatalf("render en: %v", err)
	}
	enBody := enBuf.String()
	if !strings.Contains(enBody, "Configs") {
		t.Fatalf("want English title Configs in %s", enBody)
	}
	if !strings.Contains(enBody, "Operational overview") {
		t.Fatalf("want English subtitle in %s", enBody)
	}

	var zhBuf bytes.Buffer
	if err := partials.PageHeader(zhCtx, "nav.home", "page.home.subtitle").Render(zhCtx, &zhBuf); err != nil {
		t.Fatalf("render zh: %v", err)
	}
	zhBody := zhBuf.String()
	if !strings.Contains(zhBody, "首页") {
		t.Fatalf("want Chinese title 首页 in %s", zhBody)
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
			html: renderTempl(t, partials.TokenTable(i18n.DefaultContext(), nil)),
		},
		{
			name: "agent skill table",
			html: renderTempl(t, partials.AgentSkillTable(nil)),
		},
	}
	for _, tt := range tests {
		tt := tt
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
					Name:    "demo",
					Status:  "published",
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
					Name:    "draft-one",
					Status:  "draft",
					Enabled: false,
				},
			},
			want:   "flowbot-chip-muted",
			absent: "badge-ghost",
		},
		{
			name: "shows triggers and steps columns",
			entries: []partials.PipelineListEntry{
				{
					Name:      "with-triggers",
					Status:    "published",
					Enabled:   true,
					StepCount: 3,
					Triggers: []partials.PipelineTriggerSummary{
						{Type: "event", Label: "Event: bookmark.created", Enabled: true, Letter: "E"},
						{Type: "cron", Label: "Cron: @daily", Enabled: true, Letter: "C"},
					},
				},
			},
			want:   "pipeline-triggers-with-triggers",
			absent: "badge-success",
		},
		{
			name: "run history action links to runs page",
			entries: []partials.PipelineListEntry{
				{
					Name:   "demo-pipeline",
					Status: "draft",
				},
			},
			want:   `data-testid="pipeline-runs-link-demo-pipeline"`,
			absent: "badge-success",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			html := renderTempl(t, partials.PipelineListTable(context.Background(), tt.entries))
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
