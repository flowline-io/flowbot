package layout_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/i18n"
	"github.com/flowline-io/flowbot/pkg/views/layout"
)

func TestAuthLayout(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		wantContains []string
		wantAbsent   []string
	}{
		{
			name:         "brand and no app nav",
			wantContains: []string{"Flowbot", "Homelab data hub", `data-testid="toast-container"`},
			wantAbsent:   []string{"nav-logout", "nav-group-admin", "nav-group-system"},
		},
		{
			name:         "loads core assets",
			wantContains: []string{"htmx.min.js", "app.js?v=", "app.css?v=", "custom.css?v="},
			wantAbsent:   []string{"pipeline-editor.js", "chart.js.min.js", "tailwind-browser", "daisyui.css", "alpine.csp.min.js"},
		},
		{
			name:         "english lang",
			wantContains: []string{`lang="en"`},
			wantAbsent:   []string{"fonts.googleapis.com"},
		},
		{
			name: "language switcher without Alpine",
			wantContains: []string{
				`data-testid="lang-switcher"`,
				`data-testid="lang-switch-en"`,
				`data-testid="lang-switch-zh"`,
				`hx-post="/service/web/locale"`,
				`hx-vals='{"lang":"zh"}'`,
				`hx-include="[name='csrf_token']"`,
			},
			wantAbsent: []string{`x-data="langPicker"`, `lang === 'zh'`, "alpine.csp.min.js"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			ctx := i18n.DefaultContext()
			err := layout.Auth(ctx, "Flowbot — Login").Render(ctx, &buf)
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			body := buf.String()
			for _, w := range tt.wantContains {
				if !strings.Contains(body, w) {
					t.Fatalf("want %q in body", w)
				}
			}
			for _, w := range tt.wantAbsent {
				if strings.Contains(body, w) {
					t.Fatalf("did not want %q in body", w)
				}
			}
			if tt.name == "language switcher without Alpine" {
				enBtn := switcherButton(t, body, "lang-switch-en")
				if !strings.Contains(enBtn, "font-semibold") {
					t.Fatalf("want active en button: %s", enBtn)
				}
			}
		})
	}
}

func TestAuthLayoutChinese(t *testing.T) {
	t.Parallel()
	ctx := i18n.WithCookieLang(context.Background(), i18n.CookieZH)
	var buf bytes.Buffer
	err := layout.Auth(ctx, "Flowbot — Login").Render(ctx, &buf)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()
	if !strings.Contains(html, `lang="zh-Hans"`) {
		t.Fatalf("want zh-Hans lang attribute")
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
