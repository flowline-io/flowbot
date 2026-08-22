package i18n

import (
	"strings"

	"golang.org/x/text/language"
)

const (
	// CookieName is the browser cookie storing UI locale preference.
	CookieName = "flowbot-lang"

	// CookieEN and CookieZH are accepted cookie values.
	CookieEN = "en"
	CookieZH = "zh"
)

// ParseCookie returns a normalized cookie value (en or zh). Unknown values default to en.
func ParseCookie(raw string) string {
	switch strings.TrimSpace(raw) {
	case CookieZH:
		return CookieZH
	default:
		return CookieEN
	}
}

// TagForCookie maps a cookie value to a BCP 47 language tag.
func TagForCookie(cookie string) language.Tag {
	if ParseCookie(cookie) == CookieZH {
		return language.SimplifiedChinese
	}
	return language.English
}
