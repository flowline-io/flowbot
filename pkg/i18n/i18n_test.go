package i18n_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/i18n"
)

func TestParseCookie(t *testing.T) {
	t.Parallel()
	assert.Equal(t, i18n.CookieEN, i18n.ParseCookie(""))
	assert.Equal(t, i18n.CookieEN, i18n.ParseCookie("en"))
	assert.Equal(t, i18n.CookieZH, i18n.ParseCookie("zh"))
	assert.Equal(t, i18n.CookieEN, i18n.ParseCookie("fr"))
}

func TestTEnglish(t *testing.T) {
	t.Parallel()
	ctx := i18n.DefaultContext()
	assert.Equal(t, "Inbox", i18n.T(ctx, "nav.inbox"))
	assert.Equal(t, "Sign in", i18n.T(ctx, "auth.sign_in"))
}

func TestTChinese(t *testing.T) {
	t.Parallel()
	ctx := i18n.WithLocalizer(context.Background(), i18n.LocalizerForCookie(i18n.CookieZH))
	assert.Equal(t, "收件箱", i18n.T(ctx, "nav.inbox"))
	assert.Equal(t, "登录", i18n.T(ctx, "auth.sign_in"))
	assert.Equal(t, "zh-Hans", i18n.LangTag(ctx))
}

func TestTFallbackMissingKey(t *testing.T) {
	t.Parallel()
	ctx := i18n.WithLocalizer(context.Background(), i18n.LocalizerForCookie(i18n.CookieZH))
	assert.Equal(t, "missing.key.id", i18n.T(ctx, "missing.key.id"))
}

func TestClientJSON(t *testing.T) {
	t.Parallel()
	ctx := i18n.WithLocalizer(context.Background(), i18n.LocalizerForCookie(i18n.CookieZH))
	raw := i18n.ClientJSONString(ctx)
	require.Contains(t, raw, `"common.confirm"`)
	require.Contains(t, raw, "确认")
}

func TestTagForCookie(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "en", i18n.TagForCookie(i18n.CookieEN).String())
	assert.Equal(t, "zh-Hans", i18n.TagForCookie(i18n.CookieZH).String())
}
