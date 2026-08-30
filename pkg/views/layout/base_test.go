package layout_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/flowline-io/flowbot/pkg/i18n"
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
				assertContainsAll(t, html, []string{"htmx.min.js", "app.js?v=", "alpine.csp.min.js", "app.css?v=", "custom.css?v=", "theme-init.js"})
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
			name: "delayed global htmx progress bar",
			body: templ.NopComponent,
			check: func(t *testing.T, html string) {
				t.Helper()
				assertContainsAll(t, html, []string{
					`id="flowbot-htmx-progress"`,
					`flowbot-htmx-progress`,
					`data-testid="htmx-progress"`,
					`htmx-indicator`,
				})
				assertContainsNone(t, html, []string{
					`hx-indicator="#flowbot-htmx-progress"`,
				}, "body must not set %q — it steals htmx-request from buttons")
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
		{
			name: "command palette markup and script",
			body: templ.NopComponent,
			check: func(t *testing.T, html string) {
				t.Helper()
				assertContainsAll(t, html, []string{
					`data-testid="command-palette"`,
					`data-testid="command-palette-input"`,
					`data-testid="nav-command-palette"`,
					`command-palette.js`,
					`id="command-palette-pages"`,
				})
			},
		},
		{
			name: "language switcher and i18n script",
			body: templ.NopComponent,
			check: func(t *testing.T, html string) {
				t.Helper()
				assertContainsAll(t, html, []string{
					`id="flowbot-i18n"`,
					`data-testid="lang-switcher"`,
					`data-testid="lang-switch-en"`,
					`data-testid="lang-switch-zh"`,
					`hx-post="/service/web/locale"`,
					`hx-vals='{"lang":"zh"}'`,
					`hx-include="[name='csrf_token']"`,
				})
				assertContainsNone(t, html, []string{
					`x-data="langPicker"`,
					`lang === 'zh'`,
				}, "lang switcher must not use Alpine %q")
				enBtn := switcherButton(t, html, "lang-switch-en")
				if !strings.Contains(enBtn, "font-semibold") {
					t.Fatalf("want active en button: %s", enBtn)
				}
				zhBtn := switcherButton(t, html, "lang-switch-zh")
				if !strings.Contains(zhBtn, "text-base-content/50") {
					t.Fatalf("want inactive zh button: %s", zhBtn)
				}
			},
		},
		{
			name: "desktop navbar order is brand groups then tools then user",
			body: templ.NopComponent,
			check: func(t *testing.T, html string) {
				t.Helper()
				assertContainsAll(t, html, []string{
					`data-testid="nav-primary"`,
					`class="flowbot-nav-tools"`,
				})
				prev := -1
				for _, needle := range []string{
					`data-testid="nav-primary"`,
					`data-testid="nav-group-operate"`,
					`data-testid="nav-group-automate"`,
					`data-testid="nav-group-integrate"`,
					`data-testid="nav-group-system"`,
					`data-testid="nav-command-palette"`,
					`data-testid="nav-inbox"`,
					`data-testid="lang-switcher"`,
					`data-testid="theme-quick-toggle"`,
					`data-testid="theme-picker"`,
					`data-testid="nav-logout"`,
				} {
					i := strings.Index(html, needle)
					if i < 0 {
						t.Fatalf("want %q in navbar", needle)
					}
					if i < prev {
						t.Fatalf("navbar order: %q appeared before the previous marker", needle)
					}
					prev = i
				}
			},
		},
		{
			name: "nav badges poll each endpoint once",
			body: templ.NopComponent,
			check: func(t *testing.T, html string) {
				t.Helper()
				assertPollCount(t, html, `hx-get="/service/web/inbox-badge"`, 1)
				assertPollCount(t, html, `hx-get="/service/web/approval-badge"`, 1)
				assertPollCount(t, html, `hx-get="/service/web/session-badge"`, 1)
				assertPollCount(t, html, `hx-sync="this:abort"`, 2)
				assertContainsAll(t, html, []string{
					`id="nav-inbox-badge"`,
					`id="nav-mobile-inbox-badge"`,
					`id="nav-agent-approval-badge"`,
					`id="nav-mobile-agents-approval-badge"`,
					`id="nav-agents-approval-badge"`,
					`data-testid="nav-inbox-badge"`,
					`data-testid="nav-mobile-inbox-badge"`,
					`data-testid="nav-agent-approval-badge"`,
					`data-testid="nav-mobile-agents-approval-badge"`,
					`data-testid="nav-agents-approval-badge"`,
				})
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			ctx := i18n.DefaultContext()
			err := layout.Base(ctx, "Events").Render(templ.WithChildren(ctx, tt.body), &buf)
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			tt.check(t, buf.String())
		})
	}
}

func TestBaseLayoutChinese(t *testing.T) {
	t.Parallel()
	ctx := i18n.WithCookieLang(context.Background(), i18n.CookieZH)
	var buf bytes.Buffer
	err := layout.Base(ctx, "Events").Render(templ.WithChildren(ctx, templ.NopComponent), &buf)
	requireNoError(t, err)
	html := buf.String()
	if !strings.Contains(html, `lang="zh-Hans"`) {
		t.Fatalf("want zh-Hans lang attribute")
	}
	if !strings.Contains(html, "收件箱") {
		t.Fatalf("want Chinese inbox nav label")
	}
	zhBtn := switcherButton(t, html, "lang-switch-zh")
	if !strings.Contains(zhBtn, "font-semibold") {
		t.Fatalf("want active zh button: %s", zhBtn)
	}
	enBtn := switcherButton(t, html, "lang-switch-en")
	if !strings.Contains(enBtn, "text-base-content/50") {
		t.Fatalf("want inactive en button: %s", enBtn)
	}
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("render: %v", err)
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

func assertPollCount(t *testing.T, html, needle string, want int) {
	t.Helper()
	got := strings.Count(html, needle)
	if got != want {
		t.Fatalf("want %d %q pollers, got %d", want, needle, got)
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

func switcherButton(t *testing.T, html, testid string) string {
	t.Helper()
	marker := `data-testid="` + testid + `"`
	i := strings.Index(html, marker)
	if i < 0 {
		t.Fatalf("missing %s", testid)
	}
	start := strings.LastIndex(html[:i], "<button")
	end := strings.Index(html[i:], "</button>")
	if start < 0 || end < 0 {
		t.Fatalf("button bounds for %s", testid)
	}
	return html[start : i+end+len("</button>")]
}
