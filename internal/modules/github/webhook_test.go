package github

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webhook"
)

func TestWebhookConstants(t *testing.T) {
	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{name: "PackageWebhookID should equal package", got: PackageWebhookID, expected: "package"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.got)
		})
	}
}

func TestWebhookRules_Metadata(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "should have exactly 1 webhook rule",
			test: func(t *testing.T) {
				assert.Len(t, webhookRules, 1)
			},
		},
		{
			name: "should have correct ID",
			test: func(t *testing.T) {
				assert.Equal(t, PackageWebhookID, webhookRules[0].Id)
			},
		},
		{
			name: "should have Secret=true",
			test: func(t *testing.T) {
				assert.True(t, webhookRules[0].Secret)
			},
		},
		{
			name: "should have non-nil handler",
			test: func(t *testing.T) {
				assert.NotNil(t, webhookRules[0].Handler)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestWebhookHandler(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		event    string
		data     []byte
		wantText string
	}{
		{
			name:     "ping event returns pong",
			method:   http.MethodPost,
			event:    "ping",
			data:     []byte(`{}`),
			wantText: "pong",
		},
		{
			name:     "GET method returns error method",
			method:   http.MethodGet,
			event:    "package",
			data:     []byte(`{}`),
			wantText: "error method",
		},
		{
			name:     "missing event header returns error header",
			method:   http.MethodPost,
			event:    "",
			data:     []byte(`{}`),
			wantText: "error header",
		},
		{
			name:     "empty event header value returns error event",
			method:   http.MethodPost,
			event:    "empty_sentinel",
			data:     []byte(`{}`),
			wantText: "error event",
		},
		{
			name:     "unsupported event returns not supported",
			method:   http.MethodPost,
			event:    "push",
			data:     []byte(`{}`),
			wantText: "not supported",
		},
		{
			name:     "package event with non-latest tag returns not latest",
			method:   http.MethodPost,
			event:    "package",
			data:     []byte(`{"action":"published","package":{"id":12345,"name":"test-package","namespace":"testuser","description":"A test package","ecosystem":"docker","package_type":"container","html_url":"https://github.com/testuser/test-repo/pkgs/container/test-package","package_version":{"id":67890,"version":"1.0.0","name":"test-package:1.0.0","description":"Version 1.0.0","container_metadata":{"tag":{"name":"1.0.0","digest":"sha256:abc123"}}}}}`),
			wantText: "not latest",
		},
		{
			name:     "package event with non-published action returns not published",
			method:   http.MethodPost,
			event:    "package",
			data:     []byte(`{"action":"updated","package":{"id":12345,"name":"test-package","namespace":"testuser","description":"A test package","ecosystem":"docker","package_type":"container","html_url":"https://github.com/testuser/test-repo/pkgs/container/test-package","package_version":{"id":67890,"version":"1.0.0","name":"test-package:1.0.0","description":"Version 1.0.0","container_metadata":{"tag":{"name":"latest","digest":"sha256:abc123"}}}}}`),
			wantText: "not published",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := webhookRules[0].Handler
			var headers map[string][]string
			if tt.event == "empty_sentinel" {
				headers = map[string][]string{
					"X-Github-Event": {},
				}
			} else if tt.event != "" {
				headers = map[string][]string{
					"X-Github-Event": {tt.event},
				}
			} else {
				headers = map[string][]string{}
			}

			ctx := types.Context{
				Method:  tt.method,
				Headers: headers,
			}
			result := handler(ctx, tt.data)

			msg, ok := result.(types.TextMsg)
			require.True(t, ok)
			assert.Equal(t, tt.wantText, msg.Text)
		})
	}
}

func TestWebhookRule_ImplementsInterface(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "webhookRules[0] implements webhook.Rule interface"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var _ webhook.Rule = webhookRules[0]
		})
	}
}
