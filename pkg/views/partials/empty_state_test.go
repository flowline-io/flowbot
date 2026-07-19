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
