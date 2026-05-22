// Package auth provides authentication and authorization helpers.
package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasScope(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		scopes []string
		target string
		want   bool
	}{
		{name: "admin has all", scopes: []string{ScopeAdmin}, target: ScopeHubAppsRead, want: true},
		{name: "exact match", scopes: []string{ScopeHubAppsRead}, target: ScopeHubAppsRead, want: true},
		{name: "different scope", scopes: []string{ScopeHubAppsStatus}, target: ScopeHubAppsRead, want: false},
		{name: "empty scopes", scopes: []string{}, target: ScopeHubAppsRead, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, HasScope(tt.scopes, tt.target))
		})
	}
}

func TestExtractBearerToken(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "Bearer prefix", input: "Bearer token", want: "token"},
		{name: "bearer lowercase", input: "bearer token", want: "token"},
		{name: "no prefix", input: "token", want: "token"},
		{name: "empty string", input: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, ExtractBearerToken(tt.input))
		})
	}
}
