package route

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/samber/oops"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/postgres"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/types"
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
			app.Get("/test", Authorize(func(c fiber.Ctx) error {
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

func TestLookupAccessToken(t *testing.T) {
	tests := []struct {
		name      string
		seedPlain bool
		seedHash  bool
		raw       string
		wantErr   bool
		wantMigr  bool
	}{
		{
			name:     "finds hashed token",
			seedHash: true,
			raw:      "fb_lookup_hashed_token_value",
			wantErr:  false,
		},
		{
			name:      "migrates plaintext token to hash",
			seedPlain: true,
			raw:       "fb_lookup_plain_token_value",
			wantErr:   false,
			wantMigr:  true,
		},
		{
			name:    "missing token returns not found",
			raw:     "fb_missing_token_value",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orig := store.Database
			store.Database = postgres.NewSQLiteTestAdapter(t)
			t.Cleanup(func() { store.Database = orig })

			params := types.KV{"uid": "user-1", "scopes": []string{"admin:*"}}
			exp := time.Now().Add(time.Hour)
			ctx := context.Background()
			if tt.seedHash {
				require.NoError(t, store.Database.ParameterSet(ctx, auth.HashToken(tt.raw), params, exp))
			}
			if tt.seedPlain {
				require.NoError(t, store.Database.ParameterSet(ctx, tt.raw, params, exp))
			}

			p, err := LookupAccessToken(ctx, tt.raw)
			if tt.wantErr {
				require.Error(t, err)
				require.ErrorIs(t, err, types.ErrNotFound)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, auth.HashToken(tt.raw), p.Flag)
			kv := types.KV(p.Params)
			uid, _ := kv.String("uid")
			assert.Equal(t, "user-1", uid)

			if tt.wantMigr {
				_, plainErr := store.Database.ParameterGet(ctx, tt.raw)
				require.ErrorIs(t, plainErr, types.ErrNotFound)
				_, hashErr := store.Database.ParameterGet(ctx, auth.HashToken(tt.raw))
				require.NoError(t, hashErr)
			}
		})
	}
}

func TestDeleteAccessToken(t *testing.T) {
	tests := []struct {
		name      string
		seedPlain bool
		seedHash  bool
		raw       string
	}{
		{
			name:     "deletes hashed row",
			seedHash: true,
			raw:      "fb_delete_hashed_token",
		},
		{
			name:      "deletes plaintext row",
			seedPlain: true,
			raw:       "fb_delete_plain_token",
		},
		{
			name:      "deletes both rows when both exist",
			seedPlain: true,
			seedHash:  true,
			raw:       "fb_delete_both_token",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orig := store.Database
			store.Database = postgres.NewSQLiteTestAdapter(t)
			t.Cleanup(func() { store.Database = orig })

			params := types.KV{"uid": "user-1", "scopes": []string{"admin:*"}}
			exp := time.Now().Add(time.Hour)
			ctx := context.Background()
			if tt.seedHash {
				require.NoError(t, store.Database.ParameterSet(ctx, auth.HashToken(tt.raw), params, exp))
			}
			if tt.seedPlain {
				require.NoError(t, store.Database.ParameterSet(ctx, tt.raw, params, exp))
			}

			require.NoError(t, DeleteAccessToken(ctx, tt.raw))
			_, err := store.Database.ParameterGet(ctx, auth.HashToken(tt.raw))
			require.ErrorIs(t, err, types.ErrNotFound)
			_, err = store.Database.ParameterGet(ctx, tt.raw)
			require.ErrorIs(t, err, types.ErrNotFound)
		})
	}
}

func TestCheckAccessToken_Hashed(t *testing.T) {
	tests := []struct {
		name    string
		seed    func(context.Context, string)
		raw     string
		wantUID types.Uid
		wantOK  bool
	}{
		{
			name: "valid hashed token",
			seed: func(ctx context.Context, raw string) {
				require.NoError(t, store.Database.ParameterSet(ctx, auth.HashToken(raw), types.KV{
					"uid": "user-hashed", "scopes": []string{"admin:*"},
				}, time.Now().Add(time.Hour)))
			},
			raw:     "fb_check_hashed_ok",
			wantUID: "user-hashed",
			wantOK:  true,
		},
		{
			name: "legacy plaintext migrates and validates",
			seed: func(ctx context.Context, raw string) {
				require.NoError(t, store.Database.ParameterSet(ctx, raw, types.KV{
					"uid": "user-plain", "scopes": []string{"admin:*"},
				}, time.Now().Add(time.Hour)))
			},
			raw:     "fb_check_plain_ok",
			wantUID: "user-plain",
			wantOK:  true,
		},
		{
			name:   "missing token invalid",
			seed:   func(context.Context, string) {},
			raw:    "fb_check_missing",
			wantOK: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orig := store.Database
			store.Database = postgres.NewSQLiteTestAdapter(t)
			t.Cleanup(func() { store.Database = orig })

			tt.seed(context.Background(), tt.raw)
			uid, ok := CheckAccessToken(tt.raw)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Equal(t, tt.wantUID, uid)
			}
		})
	}
}

func TestGetAccessToken(t *testing.T) {
	tests := []struct {
		name string
		req  func() *http.Request
		want string
	}{
		{
			name: "reads X-AccessToken header",
			req: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/api", http.NoBody)
				r.Header.Set("X-AccessToken", "header-token")
				return r
			},
			want: "header-token",
		},
		{
			name: "reads Authorization Bearer token",
			req: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/api", http.NoBody)
				r.Header.Set("Authorization", "Bearer bearer-token")
				return r
			},
			want: "bearer-token",
		},
		{
			name: "reads accessToken cookie",
			req: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/api", http.NoBody)
				r.AddCookie(&http.Cookie{Name: "accessToken", Value: "cookie-token"})
				return r
			},
			want: "cookie-token",
		},
		{
			name: "ignores query accessToken",
			req: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/api?accessToken=query-token", http.NoBody)
			},
			want: "",
		},
		{
			name: "ignores form accessToken",
			req: func() *http.Request {
				r := httptest.NewRequest(http.MethodPost, "/api", strings.NewReader("accessToken=form-token"))
				r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				return r
			},
			want: "",
		},
		{
			name: "header takes precedence over cookie",
			req: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/api", http.NoBody)
				r.Header.Set("X-AccessToken", "header-wins")
				r.AddCookie(&http.Cookie{Name: "accessToken", Value: "cookie-ignored"})
				return r
			},
			want: "header-wins",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetAccessToken(tt.req())
			assert.Equal(t, tt.want, got)
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
			app.Get("/noauth", authorizeWithLevel(NoAuth, "example", "GET", func(c fiber.Ctx) error {
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

func TestAuthorize_RejectsEmptyScopes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		scopes any
		want   int
	}{
		{name: "nil scopes rejected", scopes: nil, want: 401},
		{name: "empty slice rejected", scopes: []string{}, want: 401},
		{name: "admin scope accepted", scopes: []string{"admin:*"}, want: 200},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orig := store.Database
			store.Database = postgres.NewSQLiteTestAdapter(t)
			t.Cleanup(func() { store.Database = orig })

			raw := "fb_empty_scope_" + strings.ReplaceAll(tt.name, " ", "_")
			params := types.KV{"uid": "user-1"}
			if tt.scopes != nil {
				params["scopes"] = tt.scopes
			}
			require.NoError(t, store.Database.ParameterSet(context.Background(), auth.HashToken(raw), params, time.Now().Add(time.Hour)))

			app := newTestApp()
			app.Get("/test", Authorize(func(c fiber.Ctx) error {
				return c.SendString("ok")
			}))
			hreq := httptest.NewRequest("GET", "/test", http.NoBody)
			hreq.Header.Set("X-AccessToken", raw)
			resp, err := app.Test(hreq)
			require.NoError(t, err)
			assert.Equal(t, tt.want, resp.StatusCode)
		})
	}
}

func TestRequireServiceScope(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		scopes []string
		group  string
		method string
		want   int
	}{
		{name: "karakeep read allows GET", scopes: []string{auth.ScopeServiceKarakeepRead}, group: "karakeep", method: "GET", want: 200},
		{name: "karakeep read denies POST", scopes: []string{auth.ScopeServiceKarakeepRead}, group: "karakeep", method: "POST", want: 403},
		{name: "admin allows POST", scopes: []string{auth.ScopeAdmin}, group: "karakeep", method: "POST", want: 200},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orig := store.Database
			store.Database = postgres.NewSQLiteTestAdapter(t)
			t.Cleanup(func() { store.Database = orig })

			raw := "fb_svc_scope_" + strings.ReplaceAll(tt.name, " ", "_")
			require.NoError(t, store.Database.ParameterSet(context.Background(), auth.HashToken(raw), types.KV{
				"uid": "user-1", "scopes": tt.scopes,
			}, time.Now().Add(time.Hour)))

			app := newTestApp()
			handler := Authorize(RequireServiceScope(tt.group, tt.method, func(c fiber.Ctx) error {
				return c.SendString("ok")
			}))
			if tt.method == "POST" {
				app.Post("/test", handler)
			} else {
				app.Get("/test", handler)
			}
			hreq := httptest.NewRequest(tt.method, "/test", http.NoBody)
			hreq.Header.Set("X-AccessToken", raw)
			resp, err := app.Test(hreq)
			require.NoError(t, err)
			assert.Equal(t, tt.want, resp.StatusCode)
		})
	}
}
