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

func TestHasAnyScope(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		scopes []string
		want   bool
	}{
		{name: "nil scopes", scopes: nil, want: false},
		{name: "empty slice", scopes: []string{}, want: false},
		{name: "blank only", scopes: []string{"", "  "}, want: false},
		{name: "has admin", scopes: []string{ScopeAdmin}, want: true},
		{name: "has service scope", scopes: []string{ScopeServiceKarakeepRead}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, HasAnyScope(tt.scopes))
		})
	}
}

func TestHasMinimumServiceScope(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		scopes []string
		group  string
		method string
		want   bool
	}{
		{name: "admin satisfies karakeep GET", scopes: []string{ScopeAdmin}, group: "karakeep", method: "GET", want: true},
		{name: "read satisfies karakeep GET", scopes: []string{ScopeServiceKarakeepRead}, group: "karakeep", method: "GET", want: true},
		{name: "write satisfies karakeep GET", scopes: []string{ScopeServiceKarakeepWrite}, group: "karakeep", method: "GET", want: true},
		{name: "read does not satisfy karakeep POST", scopes: []string{ScopeServiceKarakeepRead}, group: "karakeep", method: "POST", want: false},
		{name: "write satisfies karakeep POST", scopes: []string{ScopeServiceKarakeepWrite}, group: "karakeep", method: "POST", want: true},
		{name: "pipeline read for web GET", scopes: []string{ScopePipelineRead}, group: "web", method: "GET", want: true},
		{name: "pipeline run for web GET", scopes: []string{ScopePipelineRun}, group: "web", method: "GET", want: true},
		{name: "pipeline read denies web POST", scopes: []string{ScopePipelineRead}, group: "web", method: "POST", want: false},
		{name: "hub capabilities for hub GET", scopes: []string{ScopeHubCapabilitiesRead}, group: "hub", method: "GET", want: true},
		{name: "empty scopes denied", scopes: []string{}, group: "example", method: "GET", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, HasMinimumServiceScope(tt.scopes, tt.group, tt.method))
		})
	}
}

func TestMinimumServiceScope(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		group  string
		method string
		want   string
	}{
		{name: "example GET", group: "example", method: "GET", want: ScopeServiceExampleRead},
		{name: "example POST", group: "example", method: "POST", want: ScopeServiceExampleWrite},
		{name: "web GET", group: "web", method: "GET", want: ScopePipelineRead},
		{name: "web DELETE", group: "web", method: "DELETE", want: ScopePipelineRun},
		{name: "hub GET", group: "hub", method: "GET", want: ScopeHubCapabilitiesRead},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, MinimumServiceScope(tt.group, tt.method))
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
