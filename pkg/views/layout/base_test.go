package layout_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/flowline-io/flowbot/pkg/views/layout"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

func TestBaseLayout(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		body  templ.Component
		check func(t *testing.T, html string)
	}{
		{
			name: "loads core assets",
			body: templ.NopComponent,
			check: func(t *testing.T, html string) {
				t.Helper()
				for _, want := range []string{"htmx.min.js", "app.js", "alpine.min.js", "daisyui.css"} {
					if !strings.Contains(html, want) {
						t.Fatalf("want %q in body", want)
					}
				}
			},
		},
		{
			name: "alpine script follows sync alpine data page scripts",
			body: partials.HomelabRegistryScripts(),
			check: func(t *testing.T, html string) {
				t.Helper()
				pageScript := strings.Index(html, `src="/static/js/homelab-registry.js"`)
				alpine := strings.Index(html, "alpine.min.js")
				if pageScript < 0 || alpine < 0 {
					t.Fatalf("missing scripts: homelab-registry=%d alpine=%d", pageScript, alpine)
				}
				if alpine < pageScript {
					t.Fatalf("alpine.min.js must appear after homelab-registry.js so alpine:init handlers register first")
				}
				if strings.Contains(html, `homelab-registry.js" defer`) {
					t.Fatal("homelab-registry.js must load synchronously so Alpine.data registers before alpine:init")
				}
			},
		},
		{
			name: "english lang and toast container",
			body: templ.NopComponent,
			check: func(t *testing.T, html string) {
				t.Helper()
				for _, want := range []string{`lang="en"`, `data-testid="toast-container"`} {
					if !strings.Contains(html, want) {
						t.Fatalf("want %q in body", want)
					}
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := layout.Base("Events").Render(templ.WithChildren(context.Background(), tt.body), &buf)
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			tt.check(t, buf.String())
		})
	}
}
