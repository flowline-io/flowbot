package route

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types/audit"
)

func TestWarnLegacyWebhookQueryToken(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		auditor    audit.Auditor
		wantAction string
		wantReason string
		wantTarget string
		wantAudit  bool
	}{
		{
			name:      "no query token skips audit",
			url:       "/webhook/demo",
			auditor:   &mockAuditor{},
			wantAudit: false,
		},
		{
			name:       "query token records rejected audit",
			url:        "/webhook/demo?token=secret-value",
			auditor:    &mockAuditor{},
			wantAction: ActionWebhookQueryTokenDeprecated,
			wantReason: legacyWebhookQueryTokenReason,
			wantTarget: "demo-pipe",
			wantAudit:  true,
		},
		{
			name:    "nil auditor is safe",
			url:     "/webhook/demo?token=secret",
			auditor: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetAuditor(tt.auditor)
			t.Cleanup(func() { SetAuditor(nil) })

			app := fiber.New()
			app.All("/webhook/*", func(c fiber.Ctx) error {
				WarnLegacyWebhookQueryToken(c, "pipeline", "demo-pipe")
				return c.SendStatus(fiber.StatusNoContent)
			})

			req := httptest.NewRequest(http.MethodPost, tt.url, http.NoBody)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			_, _ = io.Copy(io.Discard, resp.Body)

			ma, ok := tt.auditor.(*mockAuditor)
			if !ok || ma == nil {
				return
			}
			if !tt.wantAudit {
				assert.Empty(t, ma.entries)
				return
			}
			require.Len(t, ma.entries, 1)
			assert.Equal(t, tt.wantAction, ma.entries[0].Action)
			assert.Equal(t, "pipeline", ma.entries[0].Target.Type)
			assert.Equal(t, tt.wantTarget, ma.entries[0].Target.ID)
			require.Len(t, ma.reasons, 1)
			assert.Equal(t, tt.wantReason, ma.reasons[0])
			assert.NotContains(t, strings.ToLower(tt.wantReason), "secret")
			assert.NotContains(t, ma.entries[0].Target.ID, "secret")
		})
	}
}
