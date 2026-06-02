package web

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/cache"
	pkgconfig "github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"

	_ "github.com/mattn/go-sqlite3"
)

type testStore struct {
	store.Adapter
	configs     []model.ConfigItem
	configErr   error
	setConfigFn func(uid types.Uid, topic, key string, value types.KV) error
	getConfigFn func(uid types.Uid, topic, key string) (types.KV, error)
	delConfigFn func(uid types.Uid, topic, key string) error
	paramGetFn  func(ctx context.Context, flag string) (gen.Parameter, error)
	paramSetFn  func(ctx context.Context, flag string, params types.KV, expiredAt time.Time) error
	paramDelFn  func(ctx context.Context, flag string) error
	dbClient    *store.Client // in-memory SQLite client for view handler tests
}

func (s *testStore) ListConfigs(_ context.Context, _ store.ListConfigOptions) ([]model.ConfigItem, error) {
	return s.configs, s.configErr
}
func (s *testStore) ConfigSet(_ context.Context, uid types.Uid, topic, key string, value types.KV) error {
	if s.setConfigFn != nil {
		return s.setConfigFn(uid, topic, key, value)
	}
	return nil
}
func (s *testStore) ConfigGet(_ context.Context, uid types.Uid, topic, key string) (types.KV, error) {
	if s.getConfigFn != nil {
		return s.getConfigFn(uid, topic, key)
	}
	return nil, types.ErrNotFound
}
func (s *testStore) ConfigDelete(_ context.Context, uid types.Uid, topic, key string) error {
	if s.delConfigFn != nil {
		return s.delConfigFn(uid, topic, key)
	}
	return nil
}
func (s *testStore) ParameterGet(ctx context.Context, flag string) (gen.Parameter, error) {
	if s.paramGetFn != nil {
		return s.paramGetFn(ctx, flag)
	}
	return gen.Parameter{
		ID:        1,
		Flag:      flag,
		Params:    map[string]any{"uid": "testuser", "topic": "test"},
		ExpiredAt: time.Now().Add(time.Hour),
	}, nil
}

// ParameterSet stores a parameter token with the given flag, params, and expiration.
func (s *testStore) ParameterSet(ctx context.Context, flag string, params types.KV, expiredAt time.Time) error {
	if s.paramSetFn != nil {
		return s.paramSetFn(ctx, flag, params, expiredAt)
	}
	return nil
}

// ParameterDelete deletes a parameter token by flag.
func (s *testStore) ParameterDelete(ctx context.Context, flag string) error {
	if s.paramDelFn != nil {
		return s.paramDelFn(ctx, flag)
	}
	return nil
}
func (*testStore) Open(_ pkgconfig.StoreType) error { return nil }
func (*testStore) Close() error                     { return nil }
func (*testStore) IsOpen() bool                     { return false }
func (*testStore) GetName() string                  { return "test" }
func (*testStore) Stats() any                       { return nil }
func (s *testStore) GetDB() any {
	if s.dbClient != nil {
		return s.dbClient
	}
	return nil
}

func setupTestApp() (*fiber.App, *testStore) {
	ts := &testStore{}
	store.Database = ts
	handler = moduleHandler{
		authConfig: AuthConfig{Username: "admin", Password: "admin"},
	}
	config = configType{
		Enabled: true,
		Auth:    AuthConfig{Username: "admin", Password: "admin"},
	}
	loginLimiter = nil
	app := fiber.New()
	var h moduleHandler
	h.Webservice(app)
	return app, ts
}

// setupTestAppWithRateLimiter creates a Fiber test app with an active login rate limiter.
func setupTestAppWithRateLimiter() (*fiber.App, *testStore, *mockRateLimitStore) {
	ts := &testStore{}
	store.Database = ts
	handler = moduleHandler{
		authConfig: AuthConfig{Username: "admin", Password: "admin"},
	}
	config = configType{
		Enabled: true,
		Auth:    AuthConfig{Username: "admin", Password: "admin"},
	}
	mockStore := newMockRateLimitStore()
	loginLimiter = newLoginRateLimiter(mockStore, 5, 10, cache.TTL(15*time.Minute), cache.TTL(15*time.Minute))
	app := fiber.New()
	var h moduleHandler
	h.Webservice(app)
	return app, ts, mockStore
}

// setupTestAppWithDB creates a Fiber test app wired with an in-memory SQLite
// database for tests that need real PageDataStore operations (view handlers).
// Each call opens a private in-memory database identified by t.Name().
func setupTestAppWithDB(t *testing.T) (*fiber.App, *testStore, *store.Client) {
	t.Helper()

	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	dbClient, err := gen.Open("sqlite3", "file:"+dbName+"?mode=memory&cache=shared&_fk=1")
	if err != nil {
		t.Fatalf("failed opening sqlite: %v", err)
	}
	if err := dbClient.Schema.Create(context.Background()); err != nil {
		t.Fatalf("failed creating schema: %v", err)
	}
	t.Cleanup(func() { dbClient.Close() })

	ts := &testStore{dbClient: dbClient}
	store.Database = ts
	handler = moduleHandler{
		authConfig: AuthConfig{Username: "admin", Password: "admin"},
	}
	config = configType{
		Enabled: true,
		Auth:    AuthConfig{Username: "admin", Password: "admin"},
	}
	app := fiber.New()
	var h moduleHandler
	h.Webservice(app)
	return app, ts, dbClient
}

func createTestConfig(uid, topic, key string) model.ConfigItem {
	return model.ConfigItem{ID: 1, UID: uid, Topic: topic, Key: key, Value: types.KV{"v": "test"}, CreatedAt: time.Now(), UpdatedAt: time.Now()}
}

// setupTestAppForRelations creates a Fiber test app with in-memory SQLite
// and pre-seeded resource links for relations tests.
func setupTestAppForRelations(t *testing.T, seedFn func(context.Context, *store.Client) error) (*fiber.App, *testStore, *store.Client) {
	t.Helper()
	app, ts, client := setupTestAppWithDB(t)
	if seedFn != nil {
		if err := seedFn(context.Background(), client); err != nil {
			t.Fatalf("failed to seed: %v", err)
		}
	}
	return app, ts, client
}
