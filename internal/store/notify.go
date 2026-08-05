// Package store provides database storage implementations.
package store

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/notificationrecord"
	"github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/types"
)

// ListNotifyRecordsOptions is an alias of notify.ListNotifyRecordsOptions.
type ListNotifyRecordsOptions = notify.ListNotifyRecordsOptions

// RecordParams is an alias of notify.RecordParams.
type RecordParams = notify.RecordParams

// ---------------------------------------------------------------------------
// NotifyStore
// ---------------------------------------------------------------------------

// NotifyStore provides CRUD for notification delivery records.
type NotifyStore struct {
	client *gen.Client
}

// NewNotifyStore returns a NotifyStore backed by the given Ent client.
func NewNotifyStore(client *gen.Client) *NotifyStore {
	return &NotifyStore{client: client}
}

// NotifyStoreFromDB returns a NotifyStore using the global database client.
func NotifyStoreFromDB() *NotifyStore {
	return NewNotifyStore(ClientFromDB())
}

// Record inserts a notification delivery record and returns the new row ID.
// ruleID is the matched notify rule id when known; empty when no rule applied.
// New records are unread (read_at nil) until MarkRead.
func (s *NotifyStore) Record(ctx context.Context, uid, channel, templateID, summary, status, errorMsg, ruleID string, payload map[string]any) (int64, error) {
	return s.RecordParams(ctx, RecordParams{
		UID:        uid,
		Channel:    channel,
		TemplateID: templateID,
		Summary:    summary,
		Status:     status,
		ErrorMsg:   errorMsg,
		RuleID:     ruleID,
		Payload:    payload,
	})
}

// RecordParams inserts a notification delivery record with extended fields.
func (s *NotifyStore) RecordParams(ctx context.Context, p RecordParams) (int64, error) {
	if s == nil || s.client == nil {
		return 0, nil
	}
	create := s.client.NotificationRecord.Create().
		SetUID(p.UID).
		SetChannel(p.Channel).
		SetTemplateID(p.TemplateID).
		SetRuleID(p.RuleID).
		SetSummary(p.Summary).
		SetStatus(notificationrecord.Status(p.Status)).
		SetErrorMsg(p.ErrorMsg).
		SetCorrelationID(p.CorrelationID).
		SetCreatedAt(time.Now())
	if p.Payload != nil {
		create = create.SetPayloadSnapshot(p.Payload)
	}
	if p.EscalateAt != nil {
		create = create.SetEscalateAt(*p.EscalateAt)
	}
	rec, err := create.Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("record notification: %w", err)
	}
	return rec.ID, nil
}

// ListRecords returns per-user notification records, cursor-paginated (newest first).
func (s *NotifyStore) ListRecords(ctx context.Context, uid string, opts ListNotifyRecordsOptions) ([]*gen.NotificationRecord, string, error) {
	if s == nil || s.client == nil {
		return nil, "", nil
	}
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 20
	}

	q := s.client.NotificationRecord.Query().
		Where(notificationrecord.UID(uid)).
		Order(gen.Desc(notificationrecord.FieldID)).
		Limit(opts.Limit + 1)

	if opts.Channel != "" {
		q = q.Where(notificationrecord.ChannelEQ(opts.Channel))
	}
	if opts.RuleID != "" {
		q = q.Where(notificationrecord.RuleIDEQ(opts.RuleID))
	}
	if opts.Status != "" {
		q = q.Where(notificationrecord.StatusEQ(notificationrecord.Status(opts.Status)))
	}
	if opts.UnreadOnly {
		q = q.Where(notificationrecord.ReadAtIsNil())
	}
	if opts.Cursor != "" {
		id, err := strconv.ParseInt(opts.Cursor, 10, 64)
		if err == nil {
			q = q.Where(notificationrecord.IDLT(id))
		}
	}

	records, err := q.All(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("list notification records: %w", err)
	}

	var nextCursor string
	if len(records) > opts.Limit {
		nextCursor = strconv.FormatInt(records[opts.Limit-1].ID, 10)
		records = records[:opts.Limit]
	}

	return records, nextCursor, nil
}

// CountUnread returns how many unread records exist for uid on the given channel and status.
func (s *NotifyStore) CountUnread(ctx context.Context, uid, channel, status string) (int, error) {
	if s == nil || s.client == nil {
		return 0, nil
	}
	q := s.client.NotificationRecord.Query().
		Where(
			notificationrecord.UID(uid),
			notificationrecord.ReadAtIsNil(),
		)
	if channel != "" {
		q = q.Where(notificationrecord.ChannelEQ(channel))
	}
	if status != "" {
		q = q.Where(notificationrecord.StatusEQ(notificationrecord.Status(status)))
	}
	n, err := q.Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("count unread notifications: %w", err)
	}
	return n, nil
}

// MarkRead sets read_at on the given notification records owned by uid and cancels related deferred rows.
func (s *NotifyStore) MarkRead(ctx context.Context, uid string, ids ...int64) error {
	if s == nil || s.client == nil || len(ids) == 0 {
		return nil
	}
	recs, err := s.client.NotificationRecord.Query().
		Where(
			notificationrecord.UID(uid),
			notificationrecord.IDIn(ids...),
		).
		All(ctx)
	if err != nil {
		return fmt.Errorf("load notification records for mark read: %w", err)
	}
	now := time.Now()
	_, err = s.client.NotificationRecord.Update().
		Where(
			notificationrecord.UID(uid),
			notificationrecord.IDIn(ids...),
			notificationrecord.ReadAtIsNil(),
		).
		SetReadAt(now).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("mark notification records read: %w", err)
	}
	seen := make(map[string]struct{})
	correlations := make([]string, 0)
	for _, rec := range recs {
		if rec.CorrelationID == "" {
			continue
		}
		if _, ok := seen[rec.CorrelationID]; ok {
			continue
		}
		seen[rec.CorrelationID] = struct{}{}
		correlations = append(correlations, rec.CorrelationID)
	}
	return s.CancelDeferredByCorrelations(ctx, uid, correlations...)
}

// MarkReadByCorrelation marks unread inapp (or any) records with the correlation id as read and cancels deferred.
func (s *NotifyStore) MarkReadByCorrelation(ctx context.Context, uid, correlationID string) error {
	if s == nil || s.client == nil || correlationID == "" {
		return nil
	}
	now := time.Now()
	_, err := s.client.NotificationRecord.Update().
		Where(
			notificationrecord.UID(uid),
			notificationrecord.CorrelationIDEQ(correlationID),
			notificationrecord.ReadAtIsNil(),
			notificationrecord.StatusEQ(notificationrecord.StatusSuccess),
		).
		SetReadAt(now).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("mark notification records read by correlation: %w", err)
	}
	return s.CancelDeferredByCorrelation(ctx, uid, correlationID)
}

// CancelDeferredByCorrelation sets deferred rows with the correlation id to cancelled.
func (s *NotifyStore) CancelDeferredByCorrelation(ctx context.Context, uid, correlationID string) error {
	if correlationID == "" {
		return nil
	}
	return s.CancelDeferredByCorrelations(ctx, uid, correlationID)
}

// CancelDeferredByCorrelations cancels deferred notification records for any of the given correlation ids.
func (s *NotifyStore) CancelDeferredByCorrelations(ctx context.Context, uid string, correlationIDs ...string) error {
	if s == nil || s.client == nil || len(correlationIDs) == 0 {
		return nil
	}
	_, err := s.client.NotificationRecord.Update().
		Where(
			notificationrecord.UID(uid),
			notificationrecord.CorrelationIDIn(correlationIDs...),
			notificationrecord.StatusEQ(notificationrecord.StatusDeferred),
		).
		SetStatus(notificationrecord.StatusCancelled).
		ClearEscalateAt().
		Save(ctx)
	if err != nil {
		return fmt.Errorf("cancel deferred notifications: %w", err)
	}
	return nil
}

// ListDueDeferred returns deferred records whose escalate_at is at or before now.
func (s *NotifyStore) ListDueDeferred(ctx context.Context, now time.Time, limit int) ([]*gen.NotificationRecord, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}
	records, err := s.client.NotificationRecord.Query().
		Where(
			notificationrecord.StatusEQ(notificationrecord.StatusDeferred),
			notificationrecord.EscalateAtNotNil(),
			notificationrecord.EscalateAtLTE(now),
		).
		Order(gen.Asc(notificationrecord.FieldEscalateAt)).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list due deferred notifications: %w", err)
	}
	return records, nil
}

// UpdateRecordStatus sets status (and optional error) on a record by id.
func (s *NotifyStore) UpdateRecordStatus(ctx context.Context, id int64, status, errorMsg string) error {
	if s == nil || s.client == nil {
		return nil
	}
	upd := s.client.NotificationRecord.UpdateOneID(id).
		SetStatus(notificationrecord.Status(status)).
		SetErrorMsg(errorMsg).
		ClearEscalateAt()
	if err := upd.Exec(ctx); err != nil {
		if gen.IsNotFound(err) {
			return types.ErrNotFound
		}
		return fmt.Errorf("update notification record status: %w", err)
	}
	return nil
}

// HasUnreadSuccessByCorrelation reports whether an unread success record exists for the correlation.
func (s *NotifyStore) HasUnreadSuccessByCorrelation(ctx context.Context, uid, correlationID string) (bool, error) {
	if s == nil || s.client == nil || correlationID == "" {
		return false, nil
	}
	n, err := s.client.NotificationRecord.Query().
		Where(
			notificationrecord.UID(uid),
			notificationrecord.CorrelationIDEQ(correlationID),
			notificationrecord.StatusEQ(notificationrecord.StatusSuccess),
			notificationrecord.ReadAtIsNil(),
		).
		Count(ctx)
	if err != nil {
		return false, fmt.Errorf("count unread success by correlation: %w", err)
	}
	return n > 0, nil
}

// GetRecord returns a single notification record by ID.
func (s *NotifyStore) GetRecord(ctx context.Context, id int64) (*gen.NotificationRecord, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	rec, err := s.client.NotificationRecord.Get(ctx, id)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("get notification record: %w", err)
	}
	return rec, nil
}

// DeleteOldest removes the oldest records for a user exceeding keepN.
func (s *NotifyStore) DeleteOldest(ctx context.Context, uid string, keepN int) error {
	if s == nil || s.client == nil {
		return nil
	}
	if keepN <= 0 {
		return nil
	}

	total, err := s.client.NotificationRecord.Query().
		Where(notificationrecord.UID(uid)).
		Count(ctx)
	if err != nil {
		return fmt.Errorf("count records for cleanup: %w", err)
	}
	if total <= keepN {
		return nil
	}

	excess := total - keepN
	oldest, err := s.client.NotificationRecord.Query().
		Where(notificationrecord.UID(uid)).
		Order(gen.Asc(notificationrecord.FieldCreatedAt)).
		Limit(excess).
		All(ctx)
	if err != nil {
		return fmt.Errorf("find oldest records: %w", err)
	}

	ids := make([]int64, len(oldest))
	for i, rec := range oldest {
		ids[i] = rec.ID
	}
	_, err = s.client.NotificationRecord.Delete().
		Where(notificationrecord.IDIn(ids...)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete oldest records: %w", err)
	}
	return nil
}
