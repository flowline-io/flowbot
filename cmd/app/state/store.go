// Package state manages the Admin frontend global state, including token
// persistence and authentication status checks.
// All methods depend on app.Context to work correctly with the go-app lifecycle.
package state

import "github.com/maxence-charriere/go-app/v10/pkg/app"

const tokenKey = "admin_token"

// Token retrieves the current token from LocalStorage.
func Token(ctx app.Context) string {
	var token string
	ctx.LocalStorage().Get(tokenKey, &token)
	return token
}

// SetToken persists the token to LocalStorage.
func SetToken(ctx app.Context, token string) {
	ctx.LocalStorage().Set(tokenKey, token)
}

// ClearToken removes the token from LocalStorage.
func ClearToken(ctx app.Context) {
	ctx.LocalStorage().Del(tokenKey)
}

// IsAuthenticated quickly checks whether the user is currently logged in.
func IsAuthenticated(ctx app.Context) bool {
	return Token(ctx) != ""
}
