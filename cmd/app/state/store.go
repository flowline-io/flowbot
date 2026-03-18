// Package state manages the Admin frontend global state, including token
// persistence and authentication status checks.
// All methods depend on app.Context to work correctly with the go-app lifecycle.
package state

import "github.com/maxence-charriere/go-app/v10/pkg/app"

const (
	tokenKey = "admin_token"
	themeKey = "admin_theme"
)

func Token(ctx app.Context) string {
	var token string
	ctx.LocalStorage().Get(tokenKey, &token)
	return token
}

func SetToken(ctx app.Context, token string) {
	ctx.LocalStorage().Set(tokenKey, token)
}

func ClearToken(ctx app.Context) {
	ctx.LocalStorage().Del(tokenKey)
}

func IsAuthenticated(ctx app.Context) bool {
	return Token(ctx) != ""
}

func Theme(ctx app.Context) string {
	var theme string
	ctx.LocalStorage().Get(themeKey, &theme)
	if theme == "" {
		theme = "light"
	}
	return theme
}

func SetTheme(ctx app.Context, theme string) {
	ctx.LocalStorage().Set(themeKey, theme)
	if doc := app.Window().Get("document"); !doc.IsNull() {
		if html := doc.Get("documentElement"); !html.IsNull() {
			html.Call("setAttribute", "data-theme", theme)
		}
	}
	ctx.NewAction("theme-changed", app.T("theme", theme))
}

func IsDarkMode(ctx app.Context) bool {
	return Theme(ctx) == "dark"
}

func ToggleTheme(ctx app.Context) {
	if IsDarkMode(ctx) {
		SetTheme(ctx, "light")
	} else {
		SetTheme(ctx, "dark")
	}
}
