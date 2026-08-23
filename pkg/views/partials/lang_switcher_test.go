package partials_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/i18n"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

func TestLangSwitcherEnglishActive(t *testing.T) {
	t.Parallel()
	html := renderLangSwitcher(t, i18n.CookieEN)
	assertLangSwitcherHTMX(t, html)
	enBtn := langSwitcherButton(t, html, "lang-switch-en")
	if !strings.Contains(enBtn, "font-semibold") {
		t.Fatalf("want active en button: %s", enBtn)
	}
	if strings.Contains(enBtn, "text-base-content/50") {
		t.Fatalf("did not want inactive class on en: %s", enBtn)
	}
	zhBtn := langSwitcherButton(t, html, "lang-switch-zh")
	if !strings.Contains(zhBtn, "text-base-content/50") {
		t.Fatalf("want inactive zh button: %s", zhBtn)
	}
}

func TestLangSwitcherChineseActive(t *testing.T) {
	t.Parallel()
	html := renderLangSwitcher(t, i18n.CookieZH)
	assertLangSwitcherHTMX(t, html)
	zhBtn := langSwitcherButton(t, html, "lang-switch-zh")
	if !strings.Contains(zhBtn, "font-semibold") {
		t.Fatalf("want active zh button: %s", zhBtn)
	}
	enBtn := langSwitcherButton(t, html, "lang-switch-en")
	if !strings.Contains(enBtn, "text-base-content/50") {
		t.Fatalf("want inactive en button: %s", enBtn)
	}
}

func renderLangSwitcher(t *testing.T, cookie string) string {
	t.Helper()
	ctx := i18n.WithCookieLang(context.Background(), cookie)
	var buf bytes.Buffer
	if err := partials.LangSwitcher(ctx).Render(ctx, &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func assertLangSwitcherHTMX(t *testing.T, html string) {
	t.Helper()
	for _, want := range []string{
		`hx-post="/service/web/locale"`,
		`hx-vals='{"lang":"en"}'`,
		`hx-vals='{"lang":"zh"}'`,
		`hx-include="[name='csrf_token']"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("want %q in %s", want, html)
		}
	}
	for _, absent := range []string{`x-data="langPicker"`, `lang === 'zh'`} {
		if strings.Contains(html, absent) {
			t.Fatalf("did not want Alpine %q in %s", absent, html)
		}
	}
}

func langSwitcherButton(t *testing.T, html, testid string) string {
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
