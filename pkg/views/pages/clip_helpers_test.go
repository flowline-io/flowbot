package pages

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/i18n"
)

func TestClipPageTitle(t *testing.T) {
	t.Parallel()
	enCtx := i18n.DefaultContext()
	zhCtx := i18n.WithLocalizer(t.Context(), i18n.LocalizerForCookie(i18n.CookieZH))
	tests := []struct {
		name string
		d    ClipPageData
		want string
	}{
		{name: "not found en", d: ClipPageData{NotFound: true}, want: "Clip not found"},
		{name: "empty title en", d: ClipPageData{}, want: "Clip"},
		{name: "with title", d: ClipPageData{Title: "Hello"}, want: "Hello"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, clipPageTitle(enCtx, tt.d))
		})
	}
	assert.Equal(t, "未找到 Clip", clipPageTitle(zhCtx, ClipPageData{NotFound: true}))
}

func TestFormatClipMeta(t *testing.T) {
	t.Parallel()
	ctx := i18n.DefaultContext()
	ts := time.Date(2026, 7, 17, 11, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		createdAt time.Time
		words     int
		want      string
	}{
		{name: "full meta", createdAt: ts, words: 629, want: "Jul 17, 2026, 11:00 AM UTC · 629 words"},
		{name: "date only", createdAt: ts, words: 0, want: "Jul 17, 2026, 11:00 AM UTC"},
		{name: "words only", createdAt: time.Time{}, words: 12, want: "12 words"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, formatClipMeta(ctx, tt.createdAt, tt.words))
		})
	}
	zhCtx := i18n.WithLocalizer(t.Context(), i18n.LocalizerForCookie(i18n.CookieZH))
	assert.Equal(t, "12 词", formatClipMeta(zhCtx, time.Time{}, 12))
}

func TestDocTitleFlowbotChinese(t *testing.T) {
	t.Parallel()
	ctx := i18n.WithLocalizer(t.Context(), i18n.LocalizerForCookie(i18n.CookieZH))
	assert.Contains(t, DocTitleFlowbot(ctx, "nav.tokens"), "令牌")
}

func TestLifeModuleShellChineseNav(t *testing.T) {
	t.Parallel()
	ctx := i18n.WithLocalizer(t.Context(), i18n.LocalizerForCookie(i18n.CookieZH))
	var buf strings.Builder
	err := LifeQuestsPage(LifeQuestsData{}).Render(ctx, &buf)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	body := buf.String()
	assert.Contains(t, body, "任务")
}
