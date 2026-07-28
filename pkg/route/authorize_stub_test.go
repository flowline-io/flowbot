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
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	pkgconfig "github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
)

// getOnlyStub implements ParameterGet but leaves the embedded Adapter nil.
type getOnlyStub struct {
	store.Adapter
}

func (getOnlyStub) Open(pkgconfig.StoreType) error { return nil }
func (getOnlyStub) Close() error                   { return nil }
func (getOnlyStub) IsOpen() bool                   { return true }
func (getOnlyStub) GetName() string                { return "get-only-stub" }
func (getOnlyStub) Stats() any                     { return nil }
func (getOnlyStub) GetDB() any                     { return nil }

func (getOnlyStub) ParameterGet(_ context.Context, flag string) (gen.Parameter, error) {
	return gen.Parameter{
		ID:   1,
		Flag: flag,
		Params: types.KV{
			"uid":    "user-1",
			"scopes": []string{"admin:*"},
		},
		ExpiredAt: time.Now().Add(time.Hour),
	}, nil
}

type getAndSetStub struct {
	getOnlyStub
}

func (getAndSetStub) ParameterSet(_ context.Context, _ string, _ types.KV, _ time.Time) error {
	return nil
}

func TestThrottledUpdateLastUsed_NilEmbeddedParameterSetPanics(t *testing.T) {
	// Not parallel: mutates package-global store.Database.
	testStoreMu.Lock()
	orig := store.Database
	store.Database = &getOnlyStub{}
	t.Cleanup(func() {
		store.Database = orig
		testStoreMu.Unlock()
	})

	assert.Panics(t, func() {
		throttledUpdateLastUsed(types.KV{"uid": "user-1"}, "token-flag", time.Now().Add(time.Hour))
	})
}

func TestAuthorize_ParameterSetNoopSucceeds(t *testing.T) {
	// Not parallel: mutates package-global store.Database.
	testStoreMu.Lock()
	orig := store.Database
	store.Database = &getAndSetStub{}
	t.Cleanup(func() {
		store.Database = orig
		testStoreMu.Unlock()
	})

	app := newTestApp()
	app.Get("/test", Authorize(func(c fiber.Ctx) error {
		return c.SendString("ok")
	}))
	hreq := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	hreq.AddCookie(&http.Cookie{Name: "accessToken", Value: "stub-token"})
	resp, err := app.Test(hreq)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
