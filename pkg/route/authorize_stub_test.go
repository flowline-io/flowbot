package route

import (
	"context"
	"maps"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestThrottledUpdateLastUsed_NilStoreNoops(t *testing.T) {
	prev := getAccessTokenStore()
	SetAccessTokenStore(nil)
	t.Cleanup(func() { SetAccessTokenStore(prev) })

	assert.NotPanics(t, func() {
		throttledUpdateLastUsed(types.KV{"uid": "user-1"}, "token-flag")
	})
}

func TestThrottledUpdateLastUsed_PreservesExpiry(t *testing.T) {
	exp := time.Now().Add(2 * time.Hour).Truncate(time.Second)
	staleUsed := time.Now().Add(-2 * time.Minute).UTC().Format(time.RFC3339Nano)
	freshUsed := time.Now().UTC().Format(time.RFC3339Nano)
	tests := []struct {
		name      string
		params    types.KV
		wantWrite bool
	}{
		{
			name:      "missing last_used_at writes and keeps expiry",
			params:    types.KV{"uid": "user-1", "scopes": []string{"admin:*"}},
			wantWrite: true,
		},
		{
			name: "stale last_used_at writes and keeps expiry",
			params: types.KV{
				"uid": "user-1", "scopes": []string{"admin:*"}, "last_used_at": staleUsed,
			},
			wantWrite: true,
		},
		{
			name: "fresh last_used_at skips write and keeps expiry",
			params: types.KV{
				"uid": "user-1", "scopes": []string{"admin:*"}, "last_used_at": freshUsed,
			},
			wantWrite: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mem := withTestAccessTokenStore(t)
			flag := auth.HashToken("last-used-token-" + tt.name)
			require.NoError(t, mem.Set(context.Background(), flag, maps.Clone(tt.params), exp))

			throttledUpdateLastUsed(maps.Clone(tt.params), flag)

			got, err := mem.Get(context.Background(), flag)
			require.NoError(t, err)
			assert.Equal(t, exp, got.ExpiredAt)
			gotUsed, ok := got.Params["last_used_at"].(string)
			if tt.wantWrite {
				require.True(t, ok)
				assert.NotEmpty(t, gotUsed)
				assert.NotEqual(t, staleUsed, gotUsed)
			} else {
				require.True(t, ok)
				assert.Equal(t, freshUsed, gotUsed)
			}
		})
	}
}

func TestAuthorize_ParameterSetNoopSucceeds(t *testing.T) {
	mem := withTestAccessTokenStore(t)
	token := "stub-token"
	require.NoError(t, mem.Set(context.Background(), auth.HashToken(token), types.KV{
		"uid":    "user-1",
		"scopes": []string{"admin:*"},
	}, time.Now().Add(time.Hour)))

	app := newTestApp()
	app.Get("/test", Authorize(func(c fiber.Ctx) error {
		return c.SendString("ok")
	}))
	hreq := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	hreq.AddCookie(&http.Cookie{Name: "accessToken", Value: token})
	resp, err := app.Test(hreq)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
