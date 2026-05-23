package ability

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/types"
)

type stubWebhookConverterWithAuth struct {
	path     string
	verifyFn func(headers map[string]string, body []byte) error
}

func (s *stubWebhookConverterWithAuth) WebhookPath() string { return s.path }
func (s *stubWebhookConverterWithAuth) VerifySignature(headers map[string]string, body []byte) error {
	if s.verifyFn != nil {
		return s.verifyFn(headers, body)
	}
	return nil
}
func (*stubWebhookConverterWithAuth) Convert(_ []byte, _ map[string]string) ([]types.DataEvent, error) {
	return nil, nil
}

func TestWebhookHandler_NotFound(t *testing.T) {
	app := fiber.New()
	mgr := NewEventSourceManager(nil, nil, nil)
	app.Post("/webhook/provider/*", mgr.WebhookHandler())

	tests := []struct {
		name string
		path string
		want int
	}{
		{
			name: "unknown path returns 404",
			path: "/webhook/provider/unknown/hooks",
			want: fiber.StatusNotFound,
		},
		{
			name: "empty path returns 404",
			path: "/webhook/provider/",
			want: fiber.StatusNotFound,
		},
		{
			name: "trailing slash returns 404",
			path: "/webhook/provider/github/",
			want: fiber.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", tt.path, nil)
			resp, _ := app.Test(req)
			if resp.StatusCode != tt.want {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.want)
			}
		})
	}
}

func TestWebhookHandler_SignatureFail(t *testing.T) {
	app := fiber.New()
	mgr := NewEventSourceManager(nil, nil, nil)
	mgr.RegisterWebhook(&stubWebhookConverterWithAuth{
		path: "github/events",
		verifyFn: func(_ map[string]string, _ []byte) error {
			return errors.New("signature mismatch")
		},
	})
	app.Post("/webhook/provider/*", mgr.WebhookHandler())

	tests := []struct {
		name string
		body string
	}{
		{name: "invalid signature returns 401", body: `{"test": true}`},
		{name: "empty body with invalid sig", body: ``},
		{name: "large body with invalid sig", body: strings.Repeat("x", 10000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/webhook/provider/github/events",
				strings.NewReader(tt.body))
			resp, _ := app.Test(req)
			if resp.StatusCode != fiber.StatusUnauthorized {
				t.Errorf("status = %d, want %d", resp.StatusCode, fiber.StatusUnauthorized)
			}
		})
	}
}

func TestWebhookHandler_Success(t *testing.T) {
	app := fiber.New()
	mgr := NewEventSourceManager(nil, nil, nil)
	mgr.RegisterWebhook(&stubWebhookConverterWithAuth{
		path:     "github/events",
		verifyFn: nil, // nil = no error = pass
	})
	app.Post("/webhook/provider/*", mgr.WebhookHandler())

	tests := []struct {
		name string
		body string
	}{
		{name: "valid request returns 202", body: `{"action": "created"}`},
		{name: "empty body returns 202", body: ``},
		{name: "large payload returns 202", body: `{"data": "` + strings.Repeat("x", 5000) + `"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/webhook/provider/github/events",
				strings.NewReader(tt.body))
			resp, _ := app.Test(req)
			if resp.StatusCode != fiber.StatusAccepted {
				t.Errorf("status = %d, want %d", resp.StatusCode, fiber.StatusAccepted)
			}
		})
	}
}
