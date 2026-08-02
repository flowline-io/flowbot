package web

import (
	"context"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	notifypkg "github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

// testNotifyRecords adapts store.NotifyStore for web package tests (avoids importing server).
type testNotifyRecords struct{}

func (testNotifyRecords) Record(ctx context.Context, uid, channel, templateID, summary, status, errorMsg, ruleID string, payload map[string]any) (int64, error) {
	return store.NotifyStoreFromDB().Record(ctx, uid, channel, templateID, summary, status, errorMsg, ruleID, payload)
}

func (testNotifyRecords) RecordParams(ctx context.Context, p notifypkg.RecordParams) (int64, error) {
	return store.NotifyStoreFromDB().RecordParams(ctx, p)
}

func (testNotifyRecords) ListRecords(ctx context.Context, uid string, opts notifypkg.ListNotifyRecordsOptions) ([]model.NotificationRecord, string, error) {
	rows, cursor, err := store.NotifyStoreFromDB().ListRecords(ctx, uid, opts)
	if err != nil {
		return nil, "", err
	}
	return mapTestNotifyRecords(rows), cursor, nil
}

func (testNotifyRecords) CountUnread(ctx context.Context, uid, channel, status string) (int, error) {
	return store.NotifyStoreFromDB().CountUnread(ctx, uid, channel, status)
}

func (testNotifyRecords) MarkRead(ctx context.Context, uid string, ids ...int64) error {
	return store.NotifyStoreFromDB().MarkRead(ctx, uid, ids...)
}

func (testNotifyRecords) MarkReadByCorrelation(ctx context.Context, uid, correlationID string) error {
	return store.NotifyStoreFromDB().MarkReadByCorrelation(ctx, uid, correlationID)
}

func (testNotifyRecords) ListDueDeferred(ctx context.Context, now time.Time, limit int) ([]model.NotificationRecord, error) {
	rows, err := store.NotifyStoreFromDB().ListDueDeferred(ctx, now, limit)
	if err != nil {
		return nil, err
	}
	return mapTestNotifyRecords(rows), nil
}

func (testNotifyRecords) UpdateRecordStatus(ctx context.Context, id int64, status, errorMsg string) error {
	return store.NotifyStoreFromDB().UpdateRecordStatus(ctx, id, status, errorMsg)
}

func (testNotifyRecords) HasUnreadSuccessByCorrelation(ctx context.Context, uid, correlationID string) (bool, error) {
	return store.NotifyStoreFromDB().HasUnreadSuccessByCorrelation(ctx, uid, correlationID)
}

func (testNotifyRecords) GetRecord(ctx context.Context, id int64) (*model.NotificationRecord, error) {
	row, err := store.NotifyStoreFromDB().GetRecord(ctx, id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	rec := mapTestNotifyRecord(row)
	return &rec, nil
}

func (testNotifyRecords) DeleteOldest(ctx context.Context, uid string, keepN int) error {
	return store.NotifyStoreFromDB().DeleteOldest(ctx, uid, keepN)
}

func mapTestNotifyRecord(row *gen.NotificationRecord) model.NotificationRecord {
	if row == nil {
		return model.NotificationRecord{}
	}
	return model.NotificationRecord{
		ID:              row.ID,
		UID:             row.UID,
		Channel:         row.Channel,
		RuleID:          row.RuleID,
		TemplateID:      row.TemplateID,
		Summary:         row.Summary,
		Status:          string(row.Status),
		ErrorMsg:        row.ErrorMsg,
		PayloadSnapshot: row.PayloadSnapshot,
		CorrelationID:   row.CorrelationID,
		EscalateAt:      row.EscalateAt,
		ReadAt:          row.ReadAt,
		CreatedAt:       row.CreatedAt,
	}
}

func mapTestNotifyRecords(rows []*gen.NotificationRecord) []model.NotificationRecord {
	out := make([]model.NotificationRecord, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		out = append(out, mapTestNotifyRecord(row))
	}
	return out
}

type testNotifyConfigStore struct{}

func (testNotifyConfigStore) GetDefaultNotifyChannelRaw(ctx context.Context) (model.NotifyChannel, error) {
	return store.NotifyConfigStoreFromDB().GetDefaultNotifyChannelRaw(ctx)
}

func (testNotifyConfigStore) GetNotifyChannelByNameRaw(ctx context.Context, name string) (model.NotifyChannel, error) {
	return store.NotifyConfigStoreFromDB().GetNotifyChannelByNameRaw(ctx, name)
}

func (testNotifyConfigStore) CreateNotifyChannel(ctx context.Context, name, protocol, uri string) (int64, error) {
	return store.NotifyConfigStoreFromDB().CreateNotifyChannel(ctx, name, protocol, uri)
}

func (testNotifyConfigStore) GetDefaultNotifyTemplate(ctx context.Context) (model.NotifyTemplate, error) {
	return store.NotifyConfigStoreFromDB().GetDefaultNotifyTemplate(ctx)
}

func (testNotifyConfigStore) GetNotifyTemplateByTemplateID(ctx context.Context, templateID string) (model.NotifyTemplate, error) {
	return store.NotifyConfigStoreFromDB().GetNotifyTemplateByTemplateID(ctx, templateID)
}

func (testNotifyConfigStore) CreateNotifyTemplate(ctx context.Context, tmpl model.NotifyTemplate) (int64, error) {
	return store.NotifyConfigStoreFromDB().CreateNotifyTemplate(ctx, tmpl)
}

type testNotifyUserConfig struct{}

func (testNotifyUserConfig) ListConfigByPrefix(ctx context.Context, uid types.Uid, topic, prefix string) ([]model.ConfigItem, error) {
	items, err := store.ModuleDataStoreFromDB().ListConfigByPrefix(ctx, uid, topic, prefix)
	if err != nil {
		return nil, err
	}
	out := make([]model.ConfigItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, model.ConfigItem{
			ID: item.ID, UID: item.UID, Topic: item.Topic, Key: item.Key,
			Value: types.KV(item.Value), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
		})
	}
	return out, nil
}

func (testNotifyUserConfig) ConfigGet(ctx context.Context, uid types.Uid, topic, key string) (types.KV, error) {
	return store.ModuleDataStoreFromDB().ConfigGet(ctx, uid, topic, key)
}

type testAccessTokenStore struct{}

func (testAccessTokenStore) Get(ctx context.Context, flag string) (route.AccessToken, error) {
	if store.Database == nil {
		return route.AccessToken{}, types.ErrNotFound
	}
	p, err := store.ModuleDataStoreFromDB().ParameterGet(ctx, flag)
	if err != nil {
		return route.AccessToken{}, err
	}
	return route.AccessToken{
		ID:        p.ID,
		Flag:      p.Flag,
		Params:    p.Params,
		ExpiredAt: p.ExpiredAt,
	}, nil
}

func (testAccessTokenStore) Set(ctx context.Context, flag string, params types.KV, expiredAt time.Time) error {
	if store.Database == nil {
		return types.ErrUnavailable
	}
	return store.ModuleDataStoreFromDB().ParameterSet(ctx, flag, params, expiredAt)
}

func (testAccessTokenStore) Delete(ctx context.Context, flag string) error {
	if store.Database == nil {
		return nil
	}
	return store.ModuleDataStoreFromDB().ParameterDelete(ctx, flag)
}

func wireNotifyStoresForTest(t *testing.T) {
	t.Helper()
	prevRecs := notifypkg.GetNotifyStore()
	prevCfg := notifypkg.GetNotifyConfigStore()
	prevUser := notifypkg.GetNotifyUserConfig()
	notifypkg.SetNotifyRecords(testNotifyRecords{})
	notifypkg.SetNotifyConfigStore(testNotifyConfigStore{})
	notifypkg.SetNotifyUserConfig(testNotifyUserConfig{})
	route.SetAccessTokenStore(testAccessTokenStore{})
	t.Cleanup(func() {
		notifypkg.WaitForRecordAsyncForTest()
		notifypkg.SetNotifyRecords(prevRecs)
		notifypkg.SetNotifyConfigStore(prevCfg)
		notifypkg.SetNotifyUserConfig(prevUser)
		route.SetAccessTokenStore(nil)
	})
}
