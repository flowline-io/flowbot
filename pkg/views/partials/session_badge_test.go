package partials_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/i18n"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

func TestSessionBadge(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	ctx := i18n.DefaultContext()
	if err := partials.SessionBadge("admin").Render(ctx, &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()
	tests := []struct {
		name string
		want string
	}{
		{name: "svg has intrinsic width", want: `width="14"`},
		{name: "svg has intrinsic height", want: `height="14"`},
		{name: "shows username", want: "admin"},
		{name: "shows 24h session label", want: "24h session"},
		{name: "has session-badge test id", want: `data-testid="session-badge"`},
		{name: "has session-expires test id", want: `data-testid="session-expires"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(html, tt.want) {
				t.Fatalf("want %q in session badge, got %q", tt.want, html)
			}
		})
	}
}
