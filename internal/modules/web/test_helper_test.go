package web

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	pkgconfig "github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

type testStore struct {
	store.Adapter
	configs     []model.ConfigItem
	configErr   error
	setConfigFn func(uid types.Uid, topic, key string, value types.KV) error
	getConfigFn func(uid types.Uid, topic, key string) (types.KV, error)
	delConfigFn func(uid types.Uid, topic, key string) error
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
func (*testStore) ParameterGet(_ context.Context, _ string) (gen.Parameter, error) {
	return gen.Parameter{
		ID:        1,
		Flag:      "test-token",
		Params:    map[string]any{"uid": "testuser", "topic": "test"},
		ExpiredAt: time.Now().Add(time.Hour),
	}, nil
}
func (*testStore) Open(_ pkgconfig.StoreType) error { return nil }
func (*testStore) Close() error                     { return nil }
func (*testStore) IsOpen() bool                     { return false }
func (*testStore) GetName() string                  { return "test" }
func (*testStore) Stats() any                       { return nil }
func (*testStore) GetDB() any                       { return nil }

func setupTestApp() (*fiber.App, *testStore) {
	ts := &testStore{}
	store.Database = ts
	app := fiber.New()
	var h moduleHandler
	h.Webservice(app)
	return app, ts
}

func createTestConfig(uid, topic, key string) model.ConfigItem {
	return model.ConfigItem{ID: 1, UID: uid, Topic: topic, Key: key, Value: types.KV{"v": "test"}, CreatedAt: time.Now(), UpdatedAt: time.Now()}
}
