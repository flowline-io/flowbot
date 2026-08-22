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
	loc := LocalizerFromContext(ctx)
	msg, err := loc.Localize(&i18n.LocalizeConfig{MessageID: messageID})
	if err == nil && msg != "" {
		return msg
	}
	msg, err = englishFallbackLocalizer().Localize(&i18n.LocalizeConfig{MessageID: messageID})
	if err != nil {
		return messageID
	}
	return msg
}
