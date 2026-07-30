package route

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestThrottledUpdateLastUsed_NilDatabaseNoops(t *testing.T) {
	orig := store.Database
	store.Database = nil
	t.Cleanup(func() { store.Database = orig })

	assert.NotPanics(t, func() {
		throttledUpdateLastUsed(types.KV{"uid": "user-1"}, "token-flag", time.Now().Add(time.Hour))
	})
}

func TestAuthorize_ParameterSetNoopSucceeds(t *testing.T) {
	withTestStore(t)
	token := "stub-token"
	require.NoError(t, store.ModuleDataStoreFromDB().ParameterSet(context.Background(), auth.HashToken(token), types.KV{
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
