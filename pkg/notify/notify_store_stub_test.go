package notify

import (
	"context"
	"strings"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

// notifyTestStore stubs config lookups for notify gateway tests.
type notifyTestStore struct {
	store.Adapter
	configs          map[string]types.KV
	listKeys         []string
	dbClient         *store.Client
	globalChannels   map[string]model.NotifyChannel
	globalChannelErr error
}

func (s *notifyTestStore) ConfigGet(_ context.Context, _ types.Uid, _, key string) (types.KV, error) {
	if v, ok := s.configs[key]; ok {
		return v, nil
	}
	return nil, types.ErrNotFound
}

func (s *notifyTestStore) ListConfigByPrefix(_ context.Context, _ types.Uid, _, prefix string) ([]*gen.ConfigData, error) {
	var items []*gen.ConfigData
	for _, key := range s.listKeys {
		if prefix == "" || strings.HasPrefix(key, prefix) {
			items = append(items, &gen.ConfigData{Key: key})
		}
	}
	return items, nil
}

func (s *notifyTestStore) GetDB() any {
	return s.dbClient
}

func (s *notifyTestStore) GetNotifyChannelByNameRaw(_ context.Context, name string) (model.NotifyChannel, error) {
	if s.globalChannelErr != nil {
		return model.NotifyChannel{}, s.globalChannelErr
	}
	if s.globalChannels == nil {
		return model.NotifyChannel{}, types.ErrNotFound
	}
	ch, ok := s.globalChannels[name]
	if !ok {
		return model.NotifyChannel{}, types.ErrNotFound
	}
	return ch, nil
}
