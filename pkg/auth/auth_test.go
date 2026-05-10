package auth

import (
	"testing"
	"time"

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

func TestWebhookSignature(t *testing.T) {
	t.Parallel()
	now := time.Unix(1700000000, 0)
	body := []byte(`{"url":"https://example.com"}`)
	secret := "secret"
	path := "/webhook/bookmark/create"
	signature := SignWebhook(secret, "post", path, now, body)

	tests := []struct {
		name   string
		method string
		path   string
		ts     time.Time
		now    time.Time
		window time.Duration
		wantOK bool
	}{
		{
			name:   "valid signature within window",
			method: "POST",
			path:   path,
			ts:     now,
			now:    now,
			window: time.Minute,
			wantOK: true,
		},
		{
			name:   "expired timestamp",
			method: "POST",
			path:   path,
			ts:     now.Add(-2 * time.Minute),
			now:    now,
			window: time.Minute,
			wantOK: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ok := VerifyWebhookSignature(secret, tt.method, tt.path, tt.ts, body, signature, tt.now, tt.window)
			assert.Equal(t, tt.wantOK, ok)
		})
	}
}
