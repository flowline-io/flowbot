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
				assertContainsAll(t, html, []string{"htmx.min.js", "app.js", "alpine.csp.min.js", "app.css", "theme-init.js"})
				assertContainsNone(t, html, []string{"tailwind-browser", "daisyui.css"}, "did not want %q in body")
			},
		},
		{
			name:  "alpine script follows sync alpine data page scripts",
			body:  partials.HomelabRegistryScripts(),
			check: assertAlpineFollowsPageScripts,
		},
		{
			name: "english lang and toast container",
			body: templ.NopComponent,
			check: func(t *testing.T, html string) {
				t.Helper()
				assertContainsAll(t, html, []string{`lang="en"`, `data-testid="toast-container"`})
			},
		},
		{
			name: "theme picker uses static CSP-safe bindings",
			body: templ.NopComponent,
			check: func(t *testing.T, html string) {
				t.Helper()
				assertContainsAll(t, html, []string{
					`setTheme('light')`,
					`:class="theme === 'light' ? 'active' : ''"`,
					`setTheme('nord')`,
				})
				assertContainsNone(t, html, []string{
					`theme === t.id`,
					`themeClass(t.id)`,
					`x-for="t in themes"`,
				}, "CSP Alpine cannot use %q")
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

// assertContainsAll fails when html is missing any of wants.
func assertContainsAll(t *testing.T, html string, wants []string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(html, want) {
			t.Fatalf("want %q in body", want)
		}
	}
}

// assertContainsNone fails when html contains any of absents; msg must include one %q verb.
func assertContainsNone(t *testing.T, html string, absents []string, msg string) {
	t.Helper()
	for _, absent := range absents {
		if strings.Contains(html, absent) {
			t.Fatalf(msg, absent)
		}
	}
}

// assertAlpineFollowsPageScripts checks Alpine loads after sync page alpine:init scripts.
func assertAlpineFollowsPageScripts(t *testing.T, html string) {
	t.Helper()
	pageScript := strings.Index(html, `src="/static/js/homelab-registry.js"`)
	alpine := strings.Index(html, "alpine.csp.min.js")
	if pageScript < 0 || alpine < 0 {
		t.Fatalf("missing scripts: homelab-registry=%d alpine=%d", pageScript, alpine)
	}
	if alpine < pageScript {
		t.Fatalf("alpine.csp.min.js must appear after homelab-registry.js so alpine:init handlers register first")
	}
	if strings.Contains(html, `homelab-registry.js" defer`) {
		t.Fatal("homelab-registry.js must load synchronously so Alpine.data registers before alpine:init")
	}
}
