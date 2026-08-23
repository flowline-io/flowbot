package i18n

import (
	"context"
	"strings"
	"sync"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pelletier/go-toml/v2"
)

var (
	clientRawOnce sync.Once
	clientRawByID map[string]map[string]string // cookie lang -> message id -> template
)

func clientRawByLang() map[string]map[string]string {
	clientRawOnce.Do(func() {
		clientRawByID = map[string]map[string]string{
			CookieEN: {},
			CookieZH: {},
		}
		files := map[string]string{
			CookieEN: "locales/en.toml",
			CookieZH: "locales/zh.toml",
		}
		unmarshalers := map[string]i18n.UnmarshalFunc{"toml": toml.Unmarshal}
		for lang, path := range files {
			data, err := localeFS.ReadFile(path)
			if err != nil {
				continue
			}
			mf, err := i18n.ParseMessageFileBytes(data, path, unmarshalers)
			if err != nil {
				continue
			}
			for _, msg := range mf.Messages {
				if msg == nil || msg.Other == "" {
					continue
				}
				clientRawByID[lang][msg.ID] = msg.Other
			}
		}
	})
	return clientRawByID
}

// TClient returns a client-side i18n template without executing {{.Var}} placeholders.
// Client JS replaces placeholders locally via flowbotI18n + string replace.
func TClient(ctx context.Context, messageID string) string {
	lang := clientRawLang(ctx)
	if msg := clientRawMessage(lang, messageID); msg != "" {
		return msg
	}
	if lang != CookieEN {
		if msg := clientRawMessage(CookieEN, messageID); msg != "" {
			return msg
		}
	}
	return T(ctx, messageID)
}

func clientRawLang(ctx context.Context) string {
	if _, ok := ctx.Value(cookieLangKey{}).(string); ok {
		return CookieLang(ctx)
	}
	if strings.HasPrefix(LangTag(ctx), "zh") {
		return CookieZH
	}
	return CookieEN
}

func clientRawMessage(lang, messageID string) string {
	byLang, ok := clientRawByLang()[lang]
	if !ok {
		return ""
	}
	return strings.TrimSpace(byLang[messageID])
}
