package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/pipeline"
)

func makeHMACSig(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestAuthenticateWebhook(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		wcfg       *pipeline.WebhookConfig
		body       string
		setHeaders func(req *http.Request)
		wantOK     bool
	}{
		{
			name: "valid token auth returns ok",
			wcfg: &pipeline.WebhookConfig{
				Auth: pipeline.WebhookAuthConfig{Token: "secret", TokenHeader: "X-Webhook-Token"},
			},
			setHeaders: func(req *http.Request) {
				req.Header.Set("X-Webhook-Token", "secret")
			},
			wantOK: true,
		},
		{
			name: "valid HMAC auth returns ok",
			wcfg: &pipeline.WebhookConfig{
				Auth: pipeline.WebhookAuthConfig{HMACSecret: "hmac-secret", HMACHeader: "X-Hub-Signature-256"},
			},
			body: "test-body",
			setHeaders: func(req *http.Request) {
				req.Header.Set("X-Hub-Signature-256", makeHMACSig("hmac-secret", []byte("test-body")))
			},
			wantOK: true,
		},
		{
			name: "token mismatch returns unauthorized",
			wcfg: &pipeline.WebhookConfig{
				Auth: pipeline.WebhookAuthConfig{Token: "secret", TokenHeader: "X-Webhook-Token"},
			},
			setHeaders: func(req *http.Request) {
				req.Header.Set("X-Webhook-Token", "wrong")
			},
			wantOK: false,
		},
		{
			name: "HMAC mismatch returns unauthorized",
			wcfg: &pipeline.WebhookConfig{
				Auth: pipeline.WebhookAuthConfig{HMACSecret: "hmac-secret", HMACHeader: "X-Hub-Signature-256"},
			},
			body: "test-body",
			setHeaders: func(req *http.Request) {
				req.Header.Set("X-Hub-Signature-256", "sha256=deadbeef")
			},
			wantOK: false,
		},
		{
			name: "no auth configured returns unauthorized",
			wcfg: &pipeline.WebhookConfig{
				Auth: pipeline.WebhookAuthConfig{},
			},
			setHeaders: func(_ *http.Request) {},
			wantOK:     false,
		},
		{
			name:       "nil webhook config returns unauthorized",
			wcfg:       nil,
			setHeaders: func(_ *http.Request) {},
			wantOK:     false,
		},
		{
			name: "empty token header defaults to X-Webhook-Token",
			wcfg: &pipeline.WebhookConfig{
				Auth: pipeline.WebhookAuthConfig{Token: "secret", TokenHeader: ""},
			},
			setHeaders: func(req *http.Request) {
				req.Header.Set("X-Webhook-Token", "secret")
			},
			wantOK: true,
		},
		{
			name: "HMAC with uppercase signature prefix",
			wcfg: &pipeline.WebhookConfig{
				Auth: pipeline.WebhookAuthConfig{HMACSecret: "key", HMACHeader: "X-Hub-Signature-256"},
			},
			body: "payload",
			setHeaders: func(req *http.Request) {
				req.Header.Set("X-Hub-Signature-256", "SHA256="+hex.EncodeToString(hmacSum("key", "payload")))
			},
			wantOK: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := fiber.New()
			defer app.Shutdown()

			app.Post("/test-auth", func(c fiber.Ctx) error {
				status, ok := authenticateWebhook(c, tt.wcfg)
				if ok {
					return c.SendStatus(fiber.StatusOK)
				}
				return c.Status(status).SendString(http.StatusText(status))
			})

			req, err := http.NewRequest("POST", "/test-auth", strings.NewReader(tt.body))
			require.NoError(t, err)
			tt.setHeaders(req)

			resp, err := app.Test(req)
			require.NoError(t, err)

			if tt.wantOK {
				assert.Equal(t, fiber.StatusOK, resp.StatusCode)
			} else {
				assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
			}
		})
	}
}

func hmacSum(secret, body string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(body))
	return mac.Sum(nil)
}

func TestWebhookHandler_Integration(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		wcfg        *pipeline.WebhookConfig
		body        string
		contentType string
		setHeaders  func(req *http.Request)
		wantStatus  int
		wantBody    string
	}{
		{
			name: "happy path token auth mapped payload returns 202",
			wcfg: &pipeline.WebhookConfig{
				Path:      "test-cb",
				Method:    "POST",
				EventType: "test.event",
				Auth:      pipeline.WebhookAuthConfig{Token: "tok", TokenHeader: "X-Webhook-Token"},
				Payload:   "mapped",
			},
			body:        `{"key":"val"}`,
			contentType: "application/json",
			setHeaders: func(req *http.Request) {
				req.Header.Set("X-Webhook-Token", "tok")
			},
			wantStatus: fiber.StatusAccepted,
		},
		{
			name: "happy path HMAC auth raw payload returns 202",
			wcfg: &pipeline.WebhookConfig{
				Path:      "hmac-cb",
				Method:    "POST",
				EventType: "hmac.event",
				Auth:      pipeline.WebhookAuthConfig{HMACSecret: "raw-secret", HMACHeader: "X-Hub-Signature-256"},
				Payload:   "raw",
			},
			body:        "plain text",
			contentType: "text/plain",
			setHeaders: func(req *http.Request) {
				req.Header.Set("X-Hub-Signature-256", makeHMACSig("raw-secret", []byte("plain text")))
			},
			wantStatus: fiber.StatusAccepted,
		},
		{
			name: "invalid JSON in mapped mode returns 400",
			wcfg: &pipeline.WebhookConfig{
				Path:      "bad-json",
				Method:    "POST",
				EventType: "fail.event",
				Auth:      pipeline.WebhookAuthConfig{Token: "tok", TokenHeader: "X-Webhook-Token"},
				Payload:   "mapped",
			},
			body:        "not-json",
			contentType: "text/plain",
			setHeaders: func(req *http.Request) {
				req.Header.Set("X-Webhook-Token", "tok")
			},
			wantStatus: fiber.StatusBadRequest,
			wantBody:   "invalid JSON body",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := fiber.New()
			defer app.Shutdown()

			def := pipeline.Definition{
				Name:    tt.name,
				Enabled: true,
				Trigger: pipeline.Trigger{Webhook: tt.wcfg},
			}

			engine := pipeline.NewEngine([]pipeline.Definition{def}, nil, nil, nil, nil)
			defer engine.Stop()

			handler := makeWebhookHandler(engine, &def)
			app.Post("/webhook/"+tt.wcfg.Path, handler)

			req, err := http.NewRequest("POST", "/webhook/"+tt.wcfg.Path, strings.NewReader(tt.body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", tt.contentType)
			tt.setHeaders(req)

			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			if tt.wantBody != "" {
				bodyBytes, readErr := io.ReadAll(resp.Body)
				require.NoError(t, readErr)
				assert.Equal(t, tt.wantBody, string(bodyBytes))
			}
		})
	}
}

func TestWebhookHandler_AuthFailureReturns401(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		wcfg       *pipeline.WebhookConfig
		setHeaders func(req *http.Request)
	}{
		{
			name: "token mismatch in handler",
			wcfg: &pipeline.WebhookConfig{
				Path:      "auth-fail",
				Method:    "POST",
				EventType: "test.event",
				Auth:      pipeline.WebhookAuthConfig{Token: "right"},
				Payload:   "raw",
			},
			setHeaders: func(req *http.Request) {
				req.Header.Set("X-Webhook-Token", "wrong")
			},
		},
		{
			name: "no auth header provided",
			wcfg: &pipeline.WebhookConfig{
				Path:      "no-auth",
				Method:    "POST",
				EventType: "test.event",
				Auth:      pipeline.WebhookAuthConfig{Token: "secret"},
				Payload:   "raw",
			},
			setHeaders: func(_ *http.Request) {},
		},
		{
			name: "HMAC wrong in handler",
			wcfg: &pipeline.WebhookConfig{
				Path:      "hmac-fail",
				Method:    "POST",
				EventType: "test.event",
				Auth:      pipeline.WebhookAuthConfig{HMACSecret: "good-secret"},
				Payload:   "raw",
			},
			setHeaders: func(req *http.Request) {
				req.Header.Set("X-Hub-Signature-256", "sha256=00000000")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := fiber.New()
			defer app.Shutdown()

			def := pipeline.Definition{
				Name:    tt.name,
				Enabled: true,
				Trigger: pipeline.Trigger{Webhook: tt.wcfg},
			}

			engine := pipeline.NewEngine([]pipeline.Definition{def}, nil, nil, nil, nil)
			defer engine.Stop()

			handler := makeWebhookHandler(engine, &def)
			app.Post("/webhook/"+tt.wcfg.Path, handler)

			req, err := http.NewRequest("POST", "/webhook/"+tt.wcfg.Path, strings.NewReader("body"))
			require.NoError(t, err)
			tt.setHeaders(req)

			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
		})
	}
}

func TestSanitizeWebhookHeaders(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		wcfg            *pipeline.WebhookConfig
		headersToSet    map[string]string
		expectedPresent []string
		expectedAbsent  []string
	}{
		{
			name: "strips standard sensitive headers",
			headersToSet: map[string]string{
				"Authorization":       "Bearer token123",
				"X-Webhook-Token":     "secret-token",
				"X-Hub-Signature-256": "sha256=abc",
				"Cookie":              "session=abc123",
				"Content-Type":        "application/json",
				"User-Agent":          "TestAgent/1.0",
				"Accept":              "application/json",
			},
			expectedPresent: []string{"Content-Type", "User-Agent", "Accept"},
			expectedAbsent:  []string{"Authorization", "X-Webhook-Token", "X-Hub-Signature-256", "Cookie"},
		},
		{
			name: "strips configured custom auth headers",
			wcfg: &pipeline.WebhookConfig{
				Auth: pipeline.WebhookAuthConfig{
					Token:       "tok",
					TokenHeader: "X-Custom-Auth",
					HMACSecret:  "sec",
					HMACHeader:  "X-Custom-Sig",
				},
			},
			headersToSet: map[string]string{
				"X-Custom-Auth": "tok",
				"X-Custom-Sig":  "sig",
				"Content-Type":  "text/plain",
			},
			expectedPresent: []string{"Content-Type"},
			expectedAbsent:  []string{"X-Custom-Auth", "X-Custom-Sig"},
		},
		{
			name: "case-insensitive header matching",
			headersToSet: map[string]string{
				"authorization":   "Bearer token123",
				"x-webhook-token": "secret",
				"Content-Type":    "application/json",
			},
			expectedPresent: []string{"Content-Type"},
			expectedAbsent:  []string{"Authorization", "X-Webhook-Token"},
		},
		{
			name:            "nil wcfg does not panic",
			wcfg:            nil,
			headersToSet:    map[string]string{"Accept": "*/*", "Authorization": "secret"},
			expectedPresent: []string{"Accept"},
			expectedAbsent:  []string{"Authorization"},
		},
		{
			name:            "request with no headers returns empty map",
			wcfg:            &pipeline.WebhookConfig{},
			headersToSet:    map[string]string{},
			expectedPresent: nil,
			expectedAbsent:  nil,
		},
		{
			name: "custom auth header that matches a standard sensitive header",
			wcfg: &pipeline.WebhookConfig{
				Auth: pipeline.WebhookAuthConfig{
					Token:       "tok",
					TokenHeader: "Authorization",
				},
			},
			headersToSet: map[string]string{
				"Authorization": "Bearer tok",
				"Content-Type":  "application/json",
			},
			expectedPresent: []string{"Content-Type"},
			expectedAbsent:  []string{"Authorization"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app := fiber.New()
			defer app.Shutdown()

			app.Post("/test-headers", func(c fiber.Ctx) error {
				hdrs := sanitizeWebhookHeaders(c, tt.wcfg)
				for _, h := range tt.expectedPresent {
					assert.Contains(t, hdrs, h, "expected header %s to be present", h)
				}
				for _, h := range tt.expectedAbsent {
					assert.NotContains(t, hdrs, h, "expected header %s to be absent", h)
				}
				return c.SendStatus(fiber.StatusOK)
			})

			req, err := http.NewRequest("POST", "/test-headers", strings.NewReader("body"))
			require.NoError(t, err)
			for k, v := range tt.headersToSet {
				req.Header.Set(k, v)
			}

			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, fiber.StatusOK, resp.StatusCode)
		})
	}
}
