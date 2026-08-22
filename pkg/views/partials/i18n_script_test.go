package partials_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/i18n"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

func TestI18nScriptEmbedsJSON(t *testing.T) {
	t.Parallel()
	ctx := i18n.DefaultContext()
	var buf bytes.Buffer
	if err := partials.I18nScript(ctx).Render(ctx, &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()
	if strings.Contains(html, "ClientJSONString") {
		t.Fatalf("I18nScript did not interpolate JSON: %q", html)
	}
	start := strings.Index(html, ">") + 1
	end := strings.LastIndex(html, "</script>")
	if start <= 0 || end <= start {
		t.Fatalf("unexpected script markup: %q", html)
	}
	payload := strings.TrimSpace(html[start:end])
	var out map[string]string
	if err := json.Unmarshal([]byte(payload), &out); err != nil {
		t.Fatalf("invalid JSON payload %q: %v", payload, err)
	}
	if out["common.confirm"] != "Confirm" {
		t.Fatalf("common.confirm: got %q want Confirm", out["common.confirm"])
	}
	if out["confirm.default_title"] != "Confirm Action" {
		t.Fatalf("confirm.default_title: got %q want Confirm Action", out["confirm.default_title"])
	}
}

func TestI18nScriptChinese(t *testing.T) {
	t.Parallel()
	ctx := i18n.WithLocalizer(i18n.DefaultContext(), i18n.LocalizerForCookie(i18n.CookieZH))
	var buf bytes.Buffer
	if err := partials.I18nScript(ctx).Render(ctx, &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()
	start := strings.Index(html, ">") + 1
	end := strings.LastIndex(html, "</script>")
	payload := strings.TrimSpace(html[start:end])
	var out map[string]string
	if err := json.Unmarshal([]byte(payload), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if out["common.confirm"] != "确认" {
		t.Fatalf("common.confirm: got %q want 确认", out["common.confirm"])
	}
	if out["confirm.default_title"] != "确认操作" {
		t.Fatalf("confirm.default_title: got %q want 确认操作", out["confirm.default_title"])
	}
}
