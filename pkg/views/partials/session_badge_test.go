package partials_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/i18n"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

func TestSessionBadgeSVGHasIntrinsicSize(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	ctx := i18n.DefaultContext()
	if err := partials.SessionBadge("admin", "24h left").Render(ctx, &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()
	wants := []string{
		`data-testid="session-badge"`,
		`width="14"`,
		`height="14"`,
		`class="w-3.5 h-3.5"`,
		"admin",
		"24h left",
	}
	for _, want := range wants {
		if !strings.Contains(html, want) {
			t.Fatalf("want %q in session badge, got %q", want, html)
		}
	}
}
