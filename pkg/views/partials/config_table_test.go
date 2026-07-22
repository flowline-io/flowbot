package partials_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/views/partials"
)

func TestConfigTableRefreshTarget(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		want string
	}{
		{name: "root carries configs-table id for outerHTML refresh", want: `class="flowbot-surface" id="configs-table"`},
		{name: "testid on root", want: `data-testid="configs-table"`},
		{name: "rows container", want: `id="configs-rows"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			if err := partials.ConfigTable(nil).Render(context.Background(), &buf); err != nil {
				t.Fatalf("render: %v", err)
			}
			html := buf.String()
			if !strings.Contains(html, tt.want) {
				t.Fatalf("want %q in %s", tt.want, html)
			}
			if !strings.HasPrefix(strings.TrimSpace(html), `<div class="flowbot-surface" id="configs-table"`) {
				t.Fatalf("refresh target id must be on root surface, got %s", html)
			}
		})
	}
}
