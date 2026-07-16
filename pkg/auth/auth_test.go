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
		{name: "admin has metrics", scopes: []string{ScopeAdmin}, target: ScopeAdminMetrics, want: true},
		{name: "admin:metrics exact match", scopes: []string{ScopeAdminMetrics}, target: ScopeAdminMetrics, want: true},
		{name: "exact match", scopes: []string{ScopeHubAppsRead}, target: ScopeHubAppsRead, want: true},
		{name: "different scope", scopes: []string{ScopeHubAppsStatus}, target: ScopeHubAppsRead, want: false},
		{name: "metrics scope does not grant hub", scopes: []string{ScopeAdminMetrics}, target: ScopeHubAppsRead, want: false},
		{name: "empty scopes", scopes: []string{}, target: ScopeHubAppsRead, want: false},
		{name: "empty required always true", scopes: []string{}, target: "", want: true},
		{name: "legacy bookmark scope satisfies karakeep", scopes: []string{"service:bookmark:read"}, target: ScopeServiceKarakeepRead, want: true},
		{name: "legacy reader write satisfies miniflux write", scopes: []string{"service:reader:write"}, target: ScopeServiceMinifluxWrite, want: true},
		{name: "legacy kanban does not satisfy gitea", scopes: []string{"service:kanban:read"}, target: ScopeServiceGiteaRead, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, HasScope(tt.scopes, tt.target))
		})
	}
}

func TestCanonicalScope(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "maps bookmark read", input: "service:bookmark:read", want: ScopeServiceKarakeepRead},
		{name: "maps forge write", input: "service:forge:write", want: ScopeServiceGiteaWrite},
		{name: "leaves canonical unchanged", input: ScopeServiceMemosRead, want: ScopeServiceMemosRead},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, canonicalScope(tt.input))
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
