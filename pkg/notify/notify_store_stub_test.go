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
	defaultChannel   *model.NotifyChannel
	defaultTemplate  *model.NotifyTemplate
	templatesByID    map[string]model.NotifyTemplate
	createdTemplates []model.NotifyTemplate
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

func (s *notifyTestStore) GetDefaultNotifyChannelRaw(_ context.Context) (model.NotifyChannel, error) {
	if s.defaultChannel == nil {
		return model.NotifyChannel{}, types.ErrNotFound
	}
	return *s.defaultChannel, nil
}

func (s *notifyTestStore) GetDefaultNotifyTemplate(_ context.Context) (model.NotifyTemplate, error) {
	if s.defaultTemplate == nil {
		return model.NotifyTemplate{}, types.ErrNotFound
	}
	return *s.defaultTemplate, nil
}

func (s *notifyTestStore) GetNotifyTemplateByTemplateID(_ context.Context, templateID string) (model.NotifyTemplate, error) {
	if s.templatesByID != nil {
		if tmpl, ok := s.templatesByID[templateID]; ok {
			return tmpl, nil
		}
	}
	return model.NotifyTemplate{}, types.ErrNotFound
}

func (s *notifyTestStore) CreateNotifyTemplate(_ context.Context, tmpl model.NotifyTemplate) (int64, error) {
	s.createdTemplates = append(s.createdTemplates, tmpl)
	if s.templatesByID == nil {
		s.templatesByID = make(map[string]model.NotifyTemplate)
	}
	tmpl.ID = int64(len(s.createdTemplates))
	s.templatesByID[tmpl.TemplateID] = tmpl
	return tmpl.ID, nil
}
