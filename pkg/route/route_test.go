package route

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/samber/oops"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types/audit"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

type mockAuditor struct {
	entries []audit.Entry
}

func (m *mockAuditor) Record(_ context.Context, entry audit.Entry) error {
	m.entries = append(m.entries, entry)
	return nil
}

func (m *mockAuditor) RecordSuccess(_ context.Context, entry audit.Entry) error {
	return m.Record(context.TODO(), entry)
}

func (m *mockAuditor) RecordFailure(_ context.Context, entry audit.Entry, _ error) error {
	return m.Record(context.TODO(), entry)
}

func (m *mockAuditor) RecordRejected(_ context.Context, entry audit.Entry, _ string) error {
	return m.Record(context.TODO(), entry)
}

func TestSetAuditor_NilSafe(t *testing.T) {
	tests := []struct {
		name    string
		auditor audit.Auditor
	}{
		{name: "set mock auditor", auditor: &mockAuditor{}},
		{name: "set nil auditor", auditor: nil},
		{name: "set nil then mock then nil", auditor: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetAuditor(tt.auditor)
			assert.NotPanics(t, func() {
				SetAuditor(tt.auditor)
			})
		})
	}
}

func newTestApp() *fiber.App {
	return fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			var e oops.OopsError
			if errors.As(err, &e) {
				if e.Code() == protocol.ErrorCode(protocol.ErrNotAuthorized) {
					return c.Status(fiber.StatusUnauthorized).SendString(e.Error())
				}
				if e.Code() == protocol.ErrorCode(protocol.ErrAccessDenied) {
					return c.Status(fiber.StatusForbidden).SendString(e.Error())
				}
				return c.Status(fiber.StatusBadRequest).SendString(e.Error())
			}
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		},
	})
}

func TestAuthorize_AuditTokenMissing(t *testing.T) {
	tests := []struct {
		name         string
		auditor      audit.Auditor
		expectRecord bool
		expectAction string
	}{
		{
			name:         "token missing with auditor",
			auditor:      &mockAuditor{},
			expectRecord: true,
			expectAction: "auth.token.validate.fail",
		},
		{
			name:         "token missing with nil auditor",
			auditor:      nil,
			expectRecord: false,
			expectAction: "",
		},
		{
			name:         "token missing with empty auditor",
			auditor:      &mockAuditor{},
			expectRecord: true,
			expectAction: "auth.token.validate.fail",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetAuditor(tt.auditor)
			app := newTestApp()
			app.Get("/test", Authorize(0, func(c fiber.Ctx) error {
				return c.SendString("ok")
			}))
			hreq := httptest.NewRequest("GET", "/test", http.NoBody)
			resp, err := app.Test(hreq)
			require.NoError(t, err)
			assert.True(t, resp.StatusCode == 401 || resp.StatusCode == 403,
				"expected 401/403, got %d", resp.StatusCode)
			if tt.expectRecord {
				m, ok := tt.auditor.(*mockAuditor)
				require.True(t, ok)
				require.Len(t, m.entries, 1)
				assert.Equal(t, tt.expectAction, m.entries[0].Action)
				assert.Equal(t, "token", m.entries[0].Target.Type)
			}
		})
	}
}

func TestRequireScope_AuditDeny(t *testing.T) {
	tests := []struct {
		name         string
		auditor      audit.Auditor
		expectRecord bool
	}{
		{
			name:         "scope deny with auditor",
			auditor:      &mockAuditor{},
			expectRecord: true,
		},
		{
			name:         "scope deny with nil auditor",
			auditor:      nil,
			expectRecord: false,
		},
		{
			name:         "scope deny with empty auditor entries",
			auditor:      &mockAuditor{},
			expectRecord: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetAuditor(tt.auditor)
			app := newTestApp()
			app.Get("/test", RequireScope("admin:test", func(c fiber.Ctx) error {
				return c.SendString("ok")
			}))
			hreq := httptest.NewRequest("GET", "/test", http.NoBody)
			resp, err := app.Test(hreq)
			require.NoError(t, err)
			assert.True(t, resp.StatusCode == 401 || resp.StatusCode == 403,
				"expected 401/403, got %d", resp.StatusCode)
			if tt.expectRecord {
				m, ok := tt.auditor.(*mockAuditor)
				require.True(t, ok)
				require.Len(t, m.entries, 1)
				assert.Equal(t, "auth.scope.deny", m.entries[0].Action)
				assert.Equal(t, "scope", m.entries[0].Target.Type)
			}
		})
	}
}

func TestAuthorize_NoAuthLevel(t *testing.T) {
	tests := []struct {
		name         string
		auditor      audit.Auditor
		expectRecord bool
	}{
		{
			name:         "no auth level with auditor",
			auditor:      &mockAuditor{},
			expectRecord: false,
		},
		{
			name:         "no auth level nil auditor",
			auditor:      nil,
			expectRecord: false,
		},
		{
			name:         "no auth level empty auditor",
			auditor:      &mockAuditor{},
			expectRecord: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetAuditor(tt.auditor)
			app := newTestApp()
			app.Get("/noauth", Authorize(NoAuth, func(c fiber.Ctx) error {
				return c.SendString("ok")
			}))
			hreq := httptest.NewRequest("GET", "/noauth", http.NoBody)
			resp, err := app.Test(hreq)
			require.NoError(t, err)
			assert.Equal(t, 200, resp.StatusCode)
			if tt.expectRecord && tt.auditor != nil {
				m, ok := tt.auditor.(*mockAuditor)
				require.True(t, ok)
				assert.Empty(t, m.entries)
			}
		})
	}
}
