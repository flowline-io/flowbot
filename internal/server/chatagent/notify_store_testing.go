package chatagent

import (
	"context"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

type testNotifyRecords struct{}

func (testNotifyRecords) Record(ctx context.Context, uid, channel, templateID, summary, status, errorMsg, ruleID string, payload map[string]any) (int64, error) {
	return store.NotifyStoreFromDB().Record(ctx, uid, channel, templateID, summary, status, errorMsg, ruleID, payload)
}

func (testNotifyRecords) RecordParams(ctx context.Context, p notify.RecordParams) (int64, error) {
	return store.NotifyStoreFromDB().RecordParams(ctx, p)
}

func (testNotifyRecords) ListRecords(ctx context.Context, uid string, opts notify.ListNotifyRecordsOptions) ([]model.NotificationRecord, string, error) {
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

func wireNotifyStoresForTest(t *testing.T) {
	t.Helper()
	prev := notify.GetNotifyStore()
	notify.SetNotifyRecords(testNotifyRecords{})
	t.Cleanup(func() {
		WaitApprovalNotifyForTest()
		notify.WaitForRecordAsyncForTest()
		notify.SetNotifyRecords(prev)
	})
}

func clearNotifyStoresForTest(t *testing.T) {
	t.Helper()
	prev := notify.GetNotifyStore()
	notify.SetNotifyRecords(nil)
	t.Cleanup(func() {
		notify.SetNotifyRecords(prev)
	})
}
