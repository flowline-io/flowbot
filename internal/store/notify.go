// Package store provides database storage implementations.
package store

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/notificationrecord"
	"github.com/flowline-io/flowbot/pkg/types"
)

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

// ListNotifyRecordsOptions holds filters and pagination for listing notification records.
type ListNotifyRecordsOptions struct {
	Limit      int    // max 100, default 20
	Cursor     string // opaque cursor: ID value as string
	Channel    string // exact channel name filter; empty means any
	RuleID     string // exact rule_id filter; empty means any
	UnreadOnly bool   // when true, only rows with nil read_at
}

// Record inserts a notification delivery record and returns the new row ID.
// ruleID is the matched notify rule id when known; empty when no rule applied.
// New records are unread (read_at nil) until MarkRead.
func (s *NotifyStore) Record(ctx context.Context, uid, channel, templateID, summary, status, errorMsg, ruleID string, payload map[string]any) (int64, error) {
	if s == nil || s.client == nil {
		return 0, nil
	}
	create := s.client.NotificationRecord.Create().
		SetUID(uid).
		SetChannel(channel).
		SetTemplateID(templateID).
		SetRuleID(ruleID).
		SetSummary(summary).
		SetStatus(notificationrecord.Status(status)).
		SetErrorMsg(errorMsg).
		SetCreatedAt(time.Now())
	if payload != nil {
		create = create.SetPayloadSnapshot(payload)
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

// MarkRead sets read_at on the given notification records owned by uid.
func (s *NotifyStore) MarkRead(ctx context.Context, uid string, ids ...int64) error {
	if s == nil || s.client == nil || len(ids) == 0 {
		return nil
	}
	now := time.Now()
	_, err := s.client.NotificationRecord.Update().
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
	return nil
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

// LLMUsageStore persists and aggregates LLM token usage records.
