package server

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

// notifyRecordsAdapter adapts store.NotifyStore to notify.NotifyRecords.
type notifyRecordsAdapter struct{}

func (notifyRecordsAdapter) Record(ctx context.Context, uid, channel, templateID, summary, status, errorMsg, ruleID string, payload map[string]any) (int64, error) {
	return store.NotifyStoreFromDB().Record(ctx, uid, channel, templateID, summary, status, errorMsg, ruleID, payload)
}

func (notifyRecordsAdapter) RecordParams(ctx context.Context, p notify.RecordParams) (int64, error) {
	return store.NotifyStoreFromDB().RecordParams(ctx, p)
}

func (notifyRecordsAdapter) ListRecords(ctx context.Context, uid string, opts notify.ListNotifyRecordsOptions) ([]model.NotificationRecord, string, error) {
	rows, cursor, err := store.NotifyStoreFromDB().ListRecords(ctx, uid, opts)
	if err != nil {
		return nil, "", err
	}
	return mapNotifyRecords(rows), cursor, nil
}

func (notifyRecordsAdapter) CountUnread(ctx context.Context, uid, channel, status string) (int, error) {
	return store.NotifyStoreFromDB().CountUnread(ctx, uid, channel, status)
}

func (notifyRecordsAdapter) MarkRead(ctx context.Context, uid string, ids ...int64) error {
	return store.NotifyStoreFromDB().MarkRead(ctx, uid, ids...)
}

func (notifyRecordsAdapter) MarkReadByCorrelation(ctx context.Context, uid, correlationID string) error {
	return store.NotifyStoreFromDB().MarkReadByCorrelation(ctx, uid, correlationID)
}

func (notifyRecordsAdapter) ListDueDeferred(ctx context.Context, now time.Time, limit int) ([]model.NotificationRecord, error) {
	rows, err := store.NotifyStoreFromDB().ListDueDeferred(ctx, now, limit)
	if err != nil {
		return nil, err
	}
	return mapNotifyRecords(rows), nil
}

func (notifyRecordsAdapter) UpdateRecordStatus(ctx context.Context, id int64, status, errorMsg string) error {
	return store.NotifyStoreFromDB().UpdateRecordStatus(ctx, id, status, errorMsg)
}

func (notifyRecordsAdapter) HasUnreadSuccessByCorrelation(ctx context.Context, uid, correlationID string) (bool, error) {
	return store.NotifyStoreFromDB().HasUnreadSuccessByCorrelation(ctx, uid, correlationID)
}

func (notifyRecordsAdapter) GetRecord(ctx context.Context, id int64) (*model.NotificationRecord, error) {
	row, err := store.NotifyStoreFromDB().GetRecord(ctx, id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	rec := mapNotifyRecord(row)
	return &rec, nil
}

func (notifyRecordsAdapter) DeleteOldest(ctx context.Context, uid string, keepN int) error {
	return store.NotifyStoreFromDB().DeleteOldest(ctx, uid, keepN)
}

func mapNotifyRecord(row *gen.NotificationRecord) model.NotificationRecord {
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

func mapNotifyRecords(rows []*gen.NotificationRecord) []model.NotificationRecord {
	out := make([]model.NotificationRecord, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		out = append(out, mapNotifyRecord(row))
	}
	return out
}

// notifyConfigAdapter adapts store.NotifyConfigStore to notify.NotifyConfigStore.
type notifyConfigAdapter struct{}

func (notifyConfigAdapter) GetDefaultNotifyChannelRaw(ctx context.Context) (model.NotifyChannel, error) {
	return store.NotifyConfigStoreFromDB().GetDefaultNotifyChannelRaw(ctx)
}

func (notifyConfigAdapter) GetNotifyChannelByNameRaw(ctx context.Context, name string) (model.NotifyChannel, error) {
	return store.NotifyConfigStoreFromDB().GetNotifyChannelByNameRaw(ctx, name)
}

func (notifyConfigAdapter) CreateNotifyChannel(ctx context.Context, name, protocol, uri string) (int64, error) {
	return store.NotifyConfigStoreFromDB().CreateNotifyChannel(ctx, name, protocol, uri)
}

func (notifyConfigAdapter) GetDefaultNotifyTemplate(ctx context.Context) (model.NotifyTemplate, error) {
	return store.NotifyConfigStoreFromDB().GetDefaultNotifyTemplate(ctx)
}

func (notifyConfigAdapter) GetNotifyTemplateByTemplateID(ctx context.Context, templateID string) (model.NotifyTemplate, error) {
	return store.NotifyConfigStoreFromDB().GetNotifyTemplateByTemplateID(ctx, templateID)
}

func (notifyConfigAdapter) CreateNotifyTemplate(ctx context.Context, tmpl model.NotifyTemplate) (int64, error) {
	return store.NotifyConfigStoreFromDB().CreateNotifyTemplate(ctx, tmpl)
}

// notifyUserConfigAdapter adapts store.ModuleDataStore to notify.NotifyUserConfig.
type notifyUserConfigAdapter struct{}

func (notifyUserConfigAdapter) ListConfigByPrefix(ctx context.Context, uid types.Uid, topic, prefix string) ([]model.ConfigItem, error) {
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
			ID:        item.ID,
			UID:       item.UID,
			Topic:     item.Topic,
			Key:       item.Key,
			Value:     types.KV(item.Value),
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		})
	}
	return out, nil
}

func (notifyUserConfigAdapter) ConfigGet(ctx context.Context, uid types.Uid, topic, key string) (types.KV, error) {
	return store.ModuleDataStoreFromDB().ConfigGet(ctx, uid, topic, key)
}

// WireNotifyStores injects store-backed notify adapters into pkg/notify.
func WireNotifyStores() {
	notify.SetNotifyRecords(notifyRecordsAdapter{})
	notify.SetNotifyConfigStore(notifyConfigAdapter{})
	notify.SetNotifyUserConfig(notifyUserConfigAdapter{})
}
