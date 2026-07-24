package server

import (
	"context"

	storepkg "github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/capability/core"
	"github.com/flowline-io/flowbot/pkg/types"
)

// dataKVStore adapts store.Database Data* APIs to core.KVStore.
type dataKVStore struct{}

func (dataKVStore) Get(ctx context.Context, uid types.Uid, namespace, key string) (types.KV, error) {
	return storepkg.Database.DataGet(ctx, uid, namespace, key)
}

func (dataKVStore) Set(ctx context.Context, uid types.Uid, namespace, key string, value types.KV) error {
	return storepkg.Database.DataSet(ctx, uid, namespace, key, value)
}

func (dataKVStore) Delete(ctx context.Context, uid types.Uid, namespace, key string) error {
	return storepkg.Database.DataDelete(ctx, uid, namespace, key)
}

// wireCoreKVStore injects the Data-backed KV store when Database is available.
func wireCoreKVStore() {
	if storepkg.Database != nil {
		core.SetKVStore(dataKVStore{})
	}
}
