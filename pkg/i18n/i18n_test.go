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

func TestTDataTemplate(t *testing.T) {
	t.Parallel()
	ctx := i18n.DefaultContext()
	got := i18n.TData(ctx, "confirm.delete_function.message", map[string]any{"Name": "demo"})
	assert.Equal(t, "Delete function demo?", got)
}

func TestTDataChinese(t *testing.T) {
	t.Parallel()
	ctx := i18n.WithLocalizer(context.Background(), i18n.LocalizerForCookie(i18n.CookieZH))
	got := i18n.TData(ctx, "life.pager.page_of", map[string]any{"Page": 2, "Total": 5})
	assert.Equal(t, "第 2 / 5 页", got)
}

func TestTagForCookie(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "en", i18n.TagForCookie(i18n.CookieEN).String())
	assert.Equal(t, "zh-Hans", i18n.TagForCookie(i18n.CookieZH).String())
}

func TestTDefault(t *testing.T) {
	t.Parallel()
	ctx := i18n.WithLocalizer(context.Background(), i18n.LocalizerForCookie(i18n.CookieZH))
	assert.Equal(t, "English fallback", i18n.TDefault(ctx, "settings.desc.missing.path", "English fallback"))
}

func TestLeafAndPrefixedChildIDs(t *testing.T) {
	t.Parallel()
	ctx := i18n.DefaultContext()
	assert.Equal(t, "Validation error. Check your input and try again.", i18n.T(ctx, "error.validation"))
	assert.Equal(t, "state is required", i18n.T(ctx, "error.validation.state_required"))
	assert.Equal(t, "Not found. The requested resource no longer exists.", i18n.T(ctx, "error.not_found"))
	assert.Equal(t, "Agent skill file not found", i18n.T(ctx, "error.not_found.agent_skill_file"))
}
