package i18n

import (
	"context"
	"sync"

	"github.com/nicksnyder/go-i18n/v2/i18n"
)

var (
	englishLocalizerOnce sync.Once
	englishLocalizer     *i18n.Localizer
)

func englishFallbackLocalizer() *i18n.Localizer {
	englishLocalizerOnce.Do(func() {
		englishLocalizer = i18n.NewLocalizer(defaultBundle, TagForCookie(CookieEN).String())
	})
	return englishLocalizer
}

// T localizes messageID for the request context. Missing keys fall back to English.
func T(ctx context.Context, messageID string) string {
	return localize(ctx, messageID, nil)
}

// TData localizes messageID with template data (e.g. {{.Name}}). Missing keys fall back to English.
func TData(ctx context.Context, messageID string, data map[string]any) string {
	return localize(ctx, messageID, data)
}

func localize(ctx context.Context, messageID string, data map[string]any) string {
	loc := LocalizerFromContext(ctx)
	cfg := &i18n.LocalizeConfig{MessageID: messageID, TemplateData: data}
	msg, err := loc.Localize(cfg)
	if err == nil && msg != "" {
		return msg
	}
	msg, err = englishFallbackLocalizer().Localize(cfg)
	if err != nil {
		return messageID
	}
	return msg
}
