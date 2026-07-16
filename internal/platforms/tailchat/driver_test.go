package tailchat

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/config"
)

func TestVerifyTailchatWebhookToken(t *testing.T) {
	tests := []struct {
		name       string
		enabled    bool
		cfgToken   string
		header     string
		wantCode   int
		wantErrSub string
	}{
		{
			name:       "enabled with empty config rejects",
			enabled:    true,
			cfgToken:   "",
			header:     "anything",
			wantCode:   http.StatusUnauthorized,
			wantErrSub: "not configured",
		},
		{
			name:       "missing header rejects",
			enabled:    true,
			cfgToken:   "shared-secret",
			header:     "",
			wantCode:   http.StatusUnauthorized,
			wantErrSub: "missing",
		},
		{
			name:       "wrong token rejects",
			enabled:    true,
			cfgToken:   "shared-secret",
			header:     "wrong",
			wantCode:   http.StatusUnauthorized,
			wantErrSub: "invalid",
		},
		{
			name:       "matching token accepted",
			enabled:    true,
			cfgToken:   "shared-secret",
			header:     "shared-secret",
			wantCode:   http.StatusOK,
			wantErrSub: "",
		},
		{
			name:       "disabled skips token check",
			enabled:    false,
			cfgToken:   "",
			header:     "",
			wantCode:   http.StatusOK,
			wantErrSub: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := config.App.Platform.Tailchat
			t.Cleanup(func() { config.App.Platform.Tailchat = old })
			config.App.Platform.Tailchat = config.Tailchat{
				Enabled:      tt.enabled,
				WebhookToken: tt.cfgToken,
			}

			app := fiber.New(fiber.Config{
				ErrorHandler: func(c fiber.Ctx, err error) error {
					return c.Status(http.StatusUnauthorized).SendString(err.Error())
				},
			})
			d := &Driver{adapter: &Adapter{}}
			app.Post("/platform/tailchat", d.HttpServer)

			req := httptest.NewRequest(http.MethodPost, "/platform/tailchat", strings.NewReader("{}"))
			req.Header.Set("Content-Type", "application/json")
			if tt.header != "" {
				req.Header.Set(webhookTokenHeader, tt.header)
			}
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tt.wantCode, resp.StatusCode)
			if tt.wantErrSub != "" {
				body, _ := io.ReadAll(resp.Body)
				assert.Contains(t, string(body), tt.wantErrSub)
			}
		})
	}
}
