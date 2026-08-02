package module

import (
	"context"
	"sync"
	"time"

	"github.com/flowline-io/flowbot/pkg/types"
)

// FormRecord is the module-layer view of a persisted form without ORM types.
type FormRecord struct {
	ID     int64
	FormID string
	UID    string
	Topic  string
	Schema types.KV
	Values types.KV
	Extra  types.KV
	State  int
}

// BehaviorRecord is the module-layer view of a persisted behavior counter.
type BehaviorRecord struct {
	ID    int64
	UID   string
	Flag  string
	Count int32
}

// ModuleDataStore persists module form, parameter, config, and behavior data.
type ModuleDataStore interface {
	FormGet(ctx context.Context, formID string) (FormRecord, error)
	FormSet(ctx context.Context, formID string, form FormRecord) error
	ParameterSet(ctx context.Context, flag string, params types.KV, expiredAt time.Time) error
	ConfigGet(ctx context.Context, uid types.Uid, topic, key string) (types.KV, error)
	BehaviorGet(ctx context.Context, uid types.Uid, flag string) (BehaviorRecord, error)
	BehaviorIncrease(ctx context.Context, uid types.Uid, flag string, number int) error
	BehaviorSet(ctx context.Context, behavior BehaviorRecord) error
}

var (
	moduleDataStoreMu sync.RWMutex
	moduleDataStore   ModuleDataStore
)

// SetModuleDataStore wires the persistence backend used by module helpers.
func SetModuleDataStore(s ModuleDataStore) {
	moduleDataStoreMu.Lock()
	defer moduleDataStoreMu.Unlock()
	moduleDataStore = s
}

func getModuleDataStore() ModuleDataStore {
	moduleDataStoreMu.RLock()
	defer moduleDataStoreMu.RUnlock()
	return moduleDataStore
}
