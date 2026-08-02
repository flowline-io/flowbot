package server

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/types"
)

// moduleDataStore adapts store.ModuleDataStore to module.ModuleDataStore.
type moduleDataStore struct{}

func (moduleDataStore) FormGet(ctx context.Context, formID string) (module.FormRecord, error) {
	row, err := store.ModuleDataStoreFromDB().FormGet(ctx, formID)
	if err != nil {
		return module.FormRecord{}, err
	}
	return genFormToRecord(row), nil
}

func (moduleDataStore) FormSet(ctx context.Context, formID string, form module.FormRecord) error {
	return store.ModuleDataStoreFromDB().FormSet(ctx, formID, recordToGenForm(form))
}

func (moduleDataStore) ParameterSet(ctx context.Context, flag string, params types.KV, expiredAt time.Time) error {
	return store.ModuleDataStoreFromDB().ParameterSet(ctx, flag, params, expiredAt)
}

func (moduleDataStore) ConfigGet(ctx context.Context, uid types.Uid, topic, key string) (types.KV, error) {
	return store.ModuleDataStoreFromDB().ConfigGet(ctx, uid, topic, key)
}

func (moduleDataStore) BehaviorGet(ctx context.Context, uid types.Uid, flag string) (module.BehaviorRecord, error) {
	row, err := store.ModuleDataStoreFromDB().BehaviorGet(ctx, uid, flag)
	if err != nil {
		return module.BehaviorRecord{}, err
	}
	return genBehaviorToRecord(row), nil
}

func (moduleDataStore) BehaviorIncrease(ctx context.Context, uid types.Uid, flag string, number int) error {
	return store.ModuleDataStoreFromDB().BehaviorIncrease(ctx, uid, flag, number)
}

func (moduleDataStore) BehaviorSet(ctx context.Context, behavior module.BehaviorRecord) error {
	return store.ModuleDataStoreFromDB().BehaviorSet(ctx, recordToGenBehavior(behavior))
}

func genFormToRecord(f gen.Form) module.FormRecord {
	return module.FormRecord{
		ID:     f.ID,
		FormID: f.FormID,
		UID:    f.UID,
		Topic:  f.Topic,
		Schema: types.KV(f.Schema),
		Values: types.KV(f.Values),
		Extra:  types.KV(f.Extra),
		State:  f.State,
	}
}

func recordToGenForm(f module.FormRecord) gen.Form {
	return gen.Form{
		ID:     f.ID,
		FormID: f.FormID,
		UID:    f.UID,
		Topic:  f.Topic,
		Schema: map[string]any(f.Schema),
		Values: map[string]any(f.Values),
		Extra:  map[string]any(f.Extra),
		State:  f.State,
	}
}

func genBehaviorToRecord(b gen.Behavior) module.BehaviorRecord {
	return module.BehaviorRecord{
		ID:    b.ID,
		UID:   b.UID,
		Flag:  b.Flag,
		Count: b.Count,
	}
}

func recordToGenBehavior(b module.BehaviorRecord) gen.Behavior {
	return gen.Behavior{
		ID:    b.ID,
		UID:   b.UID,
		Flag:  b.Flag,
		Count: b.Count,
	}
}

// WireModuleDataStore injects the store-backed module data adapter into module.
func WireModuleDataStore() {
	module.SetModuleDataStore(moduleDataStore{})
}
