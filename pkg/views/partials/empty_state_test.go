package partials_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/views/partials"
)

func TestEmptyStateCTA(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		message      string
		detail       string
		href         string
		cta          string
		wantContains []string
		wantAbsent   []string
	}{
		{
			name:         "message only",
			message:      "Nothing here",
			wantContains: []string{"Nothing here", `data-testid="empty-state"`},
			wantAbsent:   []string{`data-testid="empty-state-cta"`},
		},
		{
			name:         "with detail",
			message:      "Empty",
			detail:       "Try again later",
			wantContains: []string{"Empty", "Try again later"},
		},
		{
			name:         "with CTA",
			message:      "No items",
			href:         "/service/web/agents",
			cta:          "Open Agents",
			wantContains: []string{"Open Agents", `href="/service/web/agents"`, `data-testid="empty-state-cta"`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := partials.EmptyStateCTA(tt.message, tt.detail, tt.href, tt.cta).Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			body := buf.String()
			for _, w := range tt.wantContains {
				if !strings.Contains(body, w) {
					t.Fatalf("want %q in %s", w, body)
				}
			}
			for _, w := range tt.wantAbsent {
				if strings.Contains(body, w) {
					t.Fatalf("did not want %q", w)
				}
			}
		})
	}
}

func TestEmptyStateHXCTA(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		message      string
		detail       string
		hxGet        string
		hxTarget     string
		hxSwap       string
		cta          string
		wantContains []string
	}{
		{
			name:         "renders hx button CTA",
			message:      "No skills yet",
			detail:       "Skills extend agents.",
			hxGet:        "/service/web/agent-skills/new",
			hxTarget:     "#agent-skills-rows",
			hxSwap:       "afterbegin",
			cta:          "Create skill",
			wantContains: []string{"No skills yet", "Skills extend agents.", "Create skill", `hx-get="/service/web/agent-skills/new"`, `hx-target="#agent-skills-rows"`, `hx-swap="afterbegin"`, `data-testid="empty-state-cta"`},
		},
		{
			name:         "omits CTA without hxGet",
			message:      "Empty",
			cta:          "Create",
			wantContains: []string{"Empty"},
		},
		{
			name:         "omits CTA without label",
			message:      "Empty",
			hxGet:        "/x",
			wantContains: []string{"Empty"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := partials.EmptyStateHXCTA(tt.message, tt.detail, tt.hxGet, tt.hxTarget, tt.hxSwap, tt.cta).Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			body := buf.String()
			for _, w := range tt.wantContains {
				if !strings.Contains(body, w) {
					t.Fatalf("want %q in %s", w, body)
				}
			}
			if tt.hxGet == "" || tt.cta == "" {
				if strings.Contains(body, `data-testid="empty-state-cta"`) {
					t.Fatalf("did not want CTA in %s", body)
				}
			}
		})
	}
}

func TestEmptyStateOnClickCTA(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		message      string
		detail       string
		onClick      string
		cta          string
		wantContains []string
	}{
		{
			name:         "renders onclick button CTA",
			message:      "No pipelines yet",
			detail:       "Create your first automation.",
			onClick:      "document.getElementById('create-modal').showModal()",
			cta:          "Create pipeline",
			wantContains: []string{"No pipelines yet", "Create pipeline", "create-modal", `data-testid="empty-state-cta"`},
		},
		{
			name:         "omits CTA without onclick",
			message:      "Empty",
			cta:          "Create",
			wantContains: []string{"Empty"},
		},
		{
			name:         "omits CTA without label",
			message:      "Empty",
			onClick:      "doThing()",
			wantContains: []string{"Empty"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := partials.EmptyStateOnClickCTA(tt.message, tt.detail, tt.onClick, tt.cta).Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			body := buf.String()
			for _, w := range tt.wantContains {
				if !strings.Contains(body, w) {
					t.Fatalf("want %q in %s", w, body)
				}
			}
			if tt.onClick == "" || tt.cta == "" {
				if strings.Contains(body, `data-testid="empty-state-cta"`) {
					t.Fatalf("did not want CTA in %s", body)
				}
			}
		})
	}
}

func TestWriteTableEmptyOOB(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		id           string
		rowsSelector string
		colspan      string
		wantContains []string
	}{
		{
			name:         "wraps empty state in oob row",
			id:           "agent-skills-empty",
			rowsSelector: "#agent-skills-rows",
			colspan:      "6",
			wantContains: []string{
				`id="agent-skills-empty"`,
				`hx-swap-oob="innerHTML:#agent-skills-rows"`,
				`colspan="6"`,
				`class="p-0"`,
				"No agent skills yet",
				"Create skill",
			},
		},
		{
			name:         "tokens empty oob",
			id:           "tokens-empty",
			rowsSelector: "#tokens-rows",
			colspan:      "7",
			wantContains: []string{`id="tokens-empty"`, `innerHTML:#tokens-rows`, `colspan="7"`},
		},
		{
			name:         "configs empty oob",
			id:           "configs-empty",
			rowsSelector: "#configs-rows",
			colspan:      "7",
			wantContains: []string{`id="configs-empty"`, "No configs yet"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			empty := partials.EmptyStateHXCTA("No agent skills yet", "Skills extend agents.", "/service/web/agent-skills/new", "#agent-skills-rows", "afterbegin", "Create skill")
			if tt.id == "configs-empty" {
				empty = partials.EmptyStateHXCTA("No configs yet", "Store module settings as key/value pairs.", "/service/web/configs/new", "#configs-rows", "afterbegin", "Create config")
			}
			if err := partials.WriteTableEmptyOOB(context.Background(), &buf, tt.id, tt.rowsSelector, tt.colspan, empty); err != nil {
				t.Fatalf("WriteTableEmptyOOB: %v", err)
			}
			body := buf.String()
			for _, w := range tt.wantContains {
				if !strings.Contains(body, w) {
					t.Fatalf("want %q in %s", w, body)
				}
			}
		})
	}
}

func TestFormError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		message string
		want    string
	}{
		{name: "renders message", message: "Name is required", want: "Name is required"},
		{name: "has test id", message: "err", want: `data-testid="form-error"`},
		{name: "alert role", message: "err", want: `role="alert"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			if err := partials.FormError(tt.message).Render(context.Background(), &buf); err != nil {
				t.Fatalf("render: %v", err)
			}
			if !strings.Contains(buf.String(), tt.want) {
				t.Fatalf("want %q", tt.want)
			}
		})
	}
}

func TestPanelSkeleton(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		want string
	}{
		{name: "testid", want: `data-testid="panel-skeleton"`},
		{name: "busy", want: `aria-busy="true"`},
		{name: "soft loader class", want: "flowbot-panel-loading"},
		{name: "spinner", want: "loading-spinner"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			if err := partials.PanelSkeleton().Render(context.Background(), &buf); err != nil {
				t.Fatalf("render: %v", err)
			}
			if !strings.Contains(buf.String(), tt.want) {
				t.Fatalf("want %q in %s", tt.want, buf.String())
			}
		})
	}
}

func TestHtmxIndicator(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		want string
	}{
		{name: "htmx indicator class", want: "htmx-indicator"},
		{name: "spinner", want: "loading-spinner"},
		{name: "extra small size", want: "loading-xs"},
		{name: "aria hidden", want: `aria-hidden="true"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			if err := partials.HtmxIndicator().Render(context.Background(), &buf); err != nil {
				t.Fatalf("render: %v", err)
			}
			if !strings.Contains(buf.String(), tt.want) {
				t.Fatalf("want %q in %s", tt.want, buf.String())
			}
		})
	}
}
