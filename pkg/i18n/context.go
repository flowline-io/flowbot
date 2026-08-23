package i18n

import (
	"context"

	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type localizerKey struct{}

type cookieLangKey struct{}

// WithLocalizer returns a context carrying loc for T and ClientJSON.
func WithLocalizer(ctx context.Context, loc *i18n.Localizer) context.Context {
	if loc == nil {
		return ctx
	}
	return context.WithValue(ctx, localizerKey{}, loc)
}

// WithCookieLang attaches a flowbot-lang cookie value and its localizer to ctx.
func WithCookieLang(ctx context.Context, cookie string) context.Context {
	cookie = ParseCookie(cookie)
	ctx = WithLocalizer(ctx, LocalizerForCookie(cookie))
	return context.WithValue(ctx, cookieLangKey{}, cookie)
}

// LocalizerFromContext returns the request localizer or an English default.
func LocalizerFromContext(ctx context.Context) *i18n.Localizer {
	if loc, ok := ctx.Value(localizerKey{}).(*i18n.Localizer); ok && loc != nil {
		return loc
	}
	return LocalizerForCookie(CookieEN)
}

// LocalizerForCookie builds a localizer for a flowbot-lang cookie value.
func LocalizerForCookie(cookie string) *i18n.Localizer {
	return i18n.NewLocalizer(defaultBundle, TagForCookie(cookie).String())
}

// DefaultContext returns a context with the English UI locale (for tests).
func DefaultContext() context.Context {
	return WithCookieLang(context.Background(), CookieEN)
}

// LangTag returns the BCP 47 tag string for html lang attributes.
func LangTag(ctx context.Context) string {
	tag := T(ctx, "_meta.lang")
	if tag == "" || tag == "_meta.lang" {
		return "en"
	}
	return tag
}

// CookieLang returns the flowbot-lang cookie value stored on ctx (en or zh).
func CookieLang(ctx context.Context) string {
	if v, ok := ctx.Value(cookieLangKey{}).(string); ok && v != "" {
		return ParseCookie(v)
	}
	return CookieEN
}
