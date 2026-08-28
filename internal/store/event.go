// Package store provides database storage implementations.
package store

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/dataevent"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/eventconsumption"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/eventoutbox"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/pipelinerun"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/pipelinesteprun"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/resourcelink"
	"github.com/flowline-io/flowbot/pkg/types"
)

// PipelineRunInfo is a lightweight view of a pipeline run for event matching display.
type PipelineRunInfo struct {
	ID            int64
	PipelineName  string
	EventID       string
	Status        string
	TriggerSource string
}

// ---------------------------------------------------------------------------
// EventStore
// ---------------------------------------------------------------------------

type EventStore struct {
	client *gen.Client
}

func NewEventStore(client *gen.Client) *EventStore {
	return &EventStore{client: client}
}

// EventStoreFromDB returns an EventStore using the global database client.
func EventStoreFromDB() *EventStore {
	return NewEventStore(ClientFromDB())
}

func (s *EventStore) AppendDataEvent(ctx context.Context, event types.DataEvent) error {
	if s == nil || s.client == nil {
		return nil
	}
	c := s.client.DataEvent.Create().
		SetEventID(event.EventID).
		SetEventType(event.EventType).
		SetSource(event.Source).
		SetCapability(event.Capability).
		SetOperation(event.Operation).
		SetApp(event.App).
		SetEntityID(event.EntityID).
		SetIdempotencyKey(event.IdempotencyKey).
		SetUID(event.UID).
		SetTopic(event.Topic).
		SetCreatedAt(time.Now())
	if event.Data != nil {
		c = c.SetData(map[string]any(event.Data))
	}
	if event.Tags != nil {
		c = c.SetTags(map[string]any(event.Tags))
	}
	_, err := c.Save(ctx)
	if err == nil && event.Source != "" {
		types.EventFilterCache.SetSource(event.Source)
	}
	if err == nil && event.EventType != "" {
		types.EventFilterCache.SetEventType(event.EventType)
	}
	return err
}

func (s *EventStore) AppendEventOutbox(ctx context.Context, event types.DataEvent) error {
	if s == nil || s.client == nil {
		return nil
	}
	payload := map[string]any{
		"event_id":        event.EventID,
		"event_type":      event.EventType,
		"source":          event.Source,
		"capability":      event.Capability,
		"operation":       event.Operation,
		"app":             event.App,
		"entity_id":       event.EntityID,
		"idempotency_key": event.IdempotencyKey,
		"uid":             event.UID,
		"topic":           event.Topic,
	}
	if event.Tags != nil {
		payload["tags"] = map[string]any(event.Tags)
	}
	if event.Data != nil {
		payload["data"] = map[string]any(event.Data)
	}
	_, err := s.client.EventOutbox.Create().
		SetEventID(event.EventID).
		SetPayload(payload).
		SetPublished(false).
		SetCreatedAt(time.Now()).
		Save(ctx)
	return err
}

func (s *EventStore) MarkOutboxPublished(ctx context.Context, eventID string) error {
	if s == nil || s.client == nil {
		return nil
	}
	n, err := s.client.EventOutbox.Update().
		Where(eventoutbox.EventID(eventID)).
		SetPublished(true).
		Save(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("event outbox mark published: event_id=%s not found", eventID)
	}
	return nil
}

// ListPendingDataEventOutbox returns unpublished DataEvent outbox rows older than olderThan.
// Skips domain-specific outbox rows (e.g. life lore) that share the same table but lack event_type.
// Scans in batches so a long run of lore rows cannot starve DataEvent redelivery.
func (s *EventStore) ListPendingDataEventOutbox(ctx context.Context, olderThan time.Time, limit int) ([]types.DataEvent, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}
	const batchSize = 100
	const maxBatches = 50
	out := make([]types.DataEvent, 0, limit)
	offset := 0
	for range maxBatches {
		rows, err := s.client.EventOutbox.Query().
			Where(
				eventoutbox.PublishedEQ(false),
				eventoutbox.CreatedAtLT(olderThan),
			).
			Order(gen.Asc(eventoutbox.FieldCreatedAt)).
			Offset(offset).
			Limit(batchSize).
			All(ctx)
		if err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			break
		}
		for _, row := range rows {
			de, ok := dataEventFromOutboxPayload(row.Payload, row.CreatedAt)
			if !ok {
				continue
			}
			if de.EventID == "" {
				de.EventID = row.EventID
			}
			out = append(out, de)
			if len(out) >= limit {
				return out, nil
			}
		}
		offset += len(rows)
		if len(rows) < batchSize {
			break
		}
	}
	return out, nil
}

// dataEventFromOutboxPayload reconstructs a DataEvent from an event_outbox payload.
// Returns ok=false for non-DataEvent rows (e.g. life.inventory.lore_requested).
func dataEventFromOutboxPayload(payload map[string]any, createdAt time.Time) (types.DataEvent, bool) {
	if payload == nil {
		return types.DataEvent{}, false
	}
	rawType, ok := payload["event_type"]
	if !ok {
		return types.DataEvent{}, false
	}
	eventType, ok := rawType.(string)
	if !ok {
		return types.DataEvent{}, false
	}
	eventType = strings.TrimSpace(eventType)
	if eventType == "" {
		return types.DataEvent{}, false
	}
	de := types.DataEvent{
		EventID:        stringField(payload, "event_id"),
		EventType:      eventType,
		Source:         stringField(payload, "source"),
		Capability:     stringField(payload, "capability"),
		Operation:      stringField(payload, "operation"),
		App:            stringField(payload, "app"),
		EntityID:       stringField(payload, "entity_id"),
		IdempotencyKey: stringField(payload, "idempotency_key"),
		UID:            stringField(payload, "uid"),
		Topic:          stringField(payload, "topic"),
		CreatedAt:      createdAt,
	}
	if tags, ok := payload["tags"].(map[string]any); ok && len(tags) > 0 {
		de.Tags = types.KV(tags)
	}
	if data, ok := payload["data"].(map[string]any); ok && len(data) > 0 {
		de.Data = types.KV(data)
	}
	return de, true
}

func stringField(m map[string]any, key string) string {
	raw, ok := m[key]
	if !ok {
		return ""
	}
	v, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(v)
}

// ListDataEventsOptions holds filters and pagination for listing data events.
type ListDataEventsOptions struct {
	Limit        int        // max 100, default 20
	Offset       int        // page offset for offset-based pagination
	Cursor       string     // opaque CreatedAt cursor (backward compatible)
	Source       string     // filter by source, empty = all
	EventType    string     // filter by event type, empty = all
	Webhook      bool       // if true, only events where data->>'_webhook_method' IS NOT NULL
	Search       string     // ILIKE match against source and data::text
	PipelineName string     // filter events that triggered a specific pipeline
	TimeStart    *time.Time // created_at >= TimeStart
	TimeEnd      *time.Time // created_at <= TimeEnd
}

// ListDataEvents returns paginated data_events ordered by created_at DESC.
// Supports offset-based pagination (when Offset > 0) and cursor-based (backward compatible).
func (s *EventStore) ListDataEvents(ctx context.Context, opts ListDataEventsOptions) ([]*gen.DataEvent, string, error) {
	if s == nil || s.client == nil {
		return nil, "", nil
	}
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 20
	}

	q := applyDataEventFilters(s.client, opts)

	// Offset-based pagination (mutually exclusive with cursor)
	if opts.Offset > 0 {
		q = q.Offset(opts.Offset).Limit(opts.Limit)
		events, err := q.All(ctx)
		if err != nil {
			return nil, "", fmt.Errorf("list data events: %w", err)
		}
		return events, "", nil
	}

	// Cursor-based pagination (backward compatible)
	q = q.Limit(opts.Limit + 1)
	if opts.Cursor != "" {
		if t, err := time.Parse("2006-01-02T15:04:05.999999Z", opts.Cursor); err == nil {
			q = q.Where(dataevent.CreatedAtLT(t))
		}
	}

	events, err := q.All(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("list data events: %w", err)
	}

	var nextCursor string
	if len(events) > opts.Limit {
		nextCursor = events[opts.Limit-1].CreatedAt.Format("2006-01-02T15:04:05.999999Z")
		events = events[:opts.Limit]
	}

	return events, nextCursor, nil
}

// applyDataEventFilters applies all filter options from ListDataEventsOptions
// to a new base query ordered by created_at DESC.
func applyDataEventFilters(client *gen.Client, opts ListDataEventsOptions) *gen.DataEventQuery {
	q := client.DataEvent.Query().
		Order(dataevent.ByCreatedAt(sql.OrderDesc()))

	if opts.Source != "" {
		q = q.Where(dataevent.Source(opts.Source))
	}
	if opts.EventType != "" {
		q = q.Where(dataevent.EventType(opts.EventType))
	}
	if opts.Webhook {
		q = q.Where(func(selector *sql.Selector) {
			selector.Where(sql.ExprP("data->>'_webhook_method' IS NOT NULL"))
		})
	}
	if opts.Search != "" {
		q = q.Where(sql.OrPredicates(
			func(s *sql.Selector) { s.Where(sql.ContainsFold("source", opts.Search)) },
			func(s *sql.Selector) {
				switch s.Dialect() {
				case dialect.Postgres:
					s.Where(sql.ExprP("CAST(data AS TEXT) ILIKE '%' || $1 || '%'", opts.Search))
				default:
					s.Where(sql.ExprP("LOWER(CAST(data AS TEXT)) LIKE LOWER('%' || $1 || '%')", opts.Search))
				}
			},
		))
	}
	if opts.PipelineName != "" {
		q = q.Where(func(s *sql.Selector) {
			s.Where(sql.ExprP(
				"event_id IN (SELECT event_id FROM pipeline_runs WHERE pipeline_name = $1)",
				opts.PipelineName,
			))
		})
	}
	if opts.TimeStart != nil {
		q = q.Where(dataevent.CreatedAtGTE(*opts.TimeStart))
	}
	if opts.TimeEnd != nil {
		q = q.Where(dataevent.CreatedAtLTE(*opts.TimeEnd))
	}

	return q
}

// CountDataEvents returns the total number of data_events matching the given filters.
// Uses the same filter predicates as ListDataEvents without pagination.
func (s *EventStore) CountDataEvents(ctx context.Context, opts ListDataEventsOptions) (int64, error) {
	if s == nil || s.client == nil {
		return 0, nil
	}

	q := applyDataEventFilters(s.client, opts)

	count, err := q.Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("count data events: %w", err)
	}

	return int64(count), nil
}

// ListDistinctEventPipelineNames returns distinct pipeline names from pipeline_runs
// that have matched events, ordered alphabetically.
func (s *EventStore) ListDistinctEventPipelineNames(ctx context.Context) ([]string, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}

	rows, err := s.client.PipelineRun.Query().
		GroupBy(pipelinerun.FieldPipelineName).
		Strings(ctx)
	if err != nil {
		return nil, fmt.Errorf("list distinct pipeline names: %w", err)
	}

	slices.Sort(rows)
	return rows, nil
}

// ListDistinctEventSources returns unique source values from data_events
// created within the given duration (e.g. 30*24*time.Hour for last 30 days).
func (s *EventStore) ListDistinctEventSources(ctx context.Context, since time.Duration) ([]string, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	sources, err := s.client.DataEvent.Query().
		Where(dataevent.CreatedAtGT(time.Now().Add(-since))).
		GroupBy(dataevent.FieldSource).
		Strings(ctx)
	if err != nil {
		return nil, fmt.Errorf("list distinct event sources: %w", err)
	}
	return sources, nil
}

// ListDistinctEventTypes returns unique event_type values from data_events
// created within the given duration.
func (s *EventStore) ListDistinctEventTypes(ctx context.Context, since time.Duration) ([]string, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	distinctTypes, err := s.client.DataEvent.Query().
		Where(dataevent.CreatedAtGT(time.Now().Add(-since))).
		GroupBy(dataevent.FieldEventType).
		Strings(ctx)
	if err != nil {
		return nil, fmt.Errorf("list distinct event types: %w", err)
	}
	return distinctTypes, nil
}

// GetDataEventByEventID looks up a single data event by its event_id.
func (s *EventStore) GetDataEventByEventID(ctx context.Context, eventID string) (*gen.DataEvent, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	e, err := s.client.DataEvent.Query().
		Where(dataevent.EventID(eventID)).
		First(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get data event by id: %w", err)
	}
	return e, nil
}

// DataEventFromRow maps a persisted data_events row to the domain event.
func DataEventFromRow(row *gen.DataEvent) types.DataEvent {
	if row == nil {
		return types.DataEvent{}
	}
	return types.DataEvent{
		EventID:        row.EventID,
		EventType:      row.EventType,
		Source:         row.Source,
		Capability:     row.Capability,
		Operation:      row.Operation,
		App:            row.App,
		EntityID:       row.EntityID,
		CreatedAt:      row.CreatedAt,
		IdempotencyKey: row.IdempotencyKey,
		UID:            row.UID,
		Topic:          row.Topic,
		Data:           types.KV(row.Data),
		Tags:           types.KV(row.Tags),
	}
}

// DeleteDataEventsOlderThan deletes data_events with created_at before cutoff
// and related history (pipeline step runs, pipeline runs, event consumptions,
// event outbox rows, and resource links that reference those events).
// Returns the number of deleted data_events rows.
func (s *EventStore) DeleteDataEventsOlderThan(ctx context.Context, cutoff time.Time) (int, error) {
	if s == nil || s.client == nil {
		return 0, nil
	}
	const batchSize = 500
	total := 0
	for {
		n, err := s.deleteDataEventsBatch(ctx, cutoff, batchSize)
		if err != nil {
			return total, err
		}
		total += n
		if n < batchSize {
			return total, nil
		}
	}
}

// deleteDataEventsBatch purges up to limit old data_events and their dependents
// in a single transaction.
func (s *EventStore) deleteDataEventsBatch(ctx context.Context, cutoff time.Time, limit int) (int, error) {
	events, err := s.client.DataEvent.Query().
		Where(dataevent.CreatedAtLT(cutoff)).
		Order(dataevent.ByCreatedAt()).
		Limit(limit).
		All(ctx)
	if err != nil {
		return 0, fmt.Errorf("list old data events: %w", err)
	}
	if len(events) == 0 {
		return 0, nil
	}

	eventIDs := make([]string, len(events))
	eventPKs := make([]int64, len(events))
	for i, e := range events {
		eventIDs[i] = e.EventID
		eventPKs[i] = e.ID
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin retention tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if err := purgeEventHistory(ctx, tx, eventIDs); err != nil {
		return 0, err
	}

	n, err := tx.DataEvent.Delete().
		Where(dataevent.IDIn(eventPKs...)).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("delete old data events: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit retention tx: %w", err)
	}
	committed = true
	return n, nil
}

// purgeEventHistory removes pipeline and delivery rows that reference eventIDs.
func purgeEventHistory(ctx context.Context, tx *gen.Tx, eventIDs []string) error {
	runs, err := tx.PipelineRun.Query().
		Where(pipelinerun.EventIDIn(eventIDs...)).
		All(ctx)
	if err != nil {
		return fmt.Errorf("list pipeline runs for retention: %w", err)
	}
	if err := purgePipelineRuns(ctx, tx, runs); err != nil {
		return err
	}
	if _, err := tx.ResourceLink.Delete().
		Where(resourcelink.Or(
			resourcelink.SourceEventIDIn(eventIDs...),
			resourcelink.TargetEventIDIn(eventIDs...),
		)).
		Exec(ctx); err != nil {
		return fmt.Errorf("delete resource links by event: %w", err)
	}
	if _, err := tx.EventConsumption.Delete().
		Where(eventconsumption.EventIDIn(eventIDs...)).
		Exec(ctx); err != nil {
		return fmt.Errorf("delete event consumptions: %w", err)
	}
	if _, err := tx.EventOutbox.Delete().
		Where(eventoutbox.EventIDIn(eventIDs...)).
		Exec(ctx); err != nil {
		return fmt.Errorf("delete event outbox: %w", err)
	}
	return nil
}

// purgePipelineRuns deletes step runs, resource links, and pipeline runs for the given runs.
func purgePipelineRuns(ctx context.Context, tx *gen.Tx, runs []*gen.PipelineRun) error {
	if len(runs) == 0 {
		return nil
	}
	runIDs := make([]int64, len(runs))
	for i, r := range runs {
		runIDs[i] = r.ID
	}
	if _, err := tx.PipelineStepRun.Delete().
		Where(pipelinesteprun.PipelineRunIDIn(runIDs...)).
		Exec(ctx); err != nil {
		return fmt.Errorf("delete pipeline step runs: %w", err)
	}
	if _, err := tx.ResourceLink.Delete().
		Where(resourcelink.PipelineRunIDIn(runIDs...)).
		Exec(ctx); err != nil {
		return fmt.Errorf("delete resource links by run: %w", err)
	}
	if _, err := tx.PipelineRun.Delete().
		Where(pipelinerun.IDIn(runIDs...)).
		Exec(ctx); err != nil {
		return fmt.Errorf("delete pipeline runs: %w", err)
	}
	return nil
}

// GetPipelineRunsForEvents batch-looks up pipeline runs for the given event IDs.
// Returns a map of eventID -> []PipelineRunInfo.
func (s *EventStore) GetPipelineRunsForEvents(ctx context.Context, eventIDs []string) (map[string][]PipelineRunInfo, error) {
	if s == nil || s.client == nil || len(eventIDs) == 0 {
		return nil, nil
	}
	runs, err := s.client.PipelineRun.Query().
		Where(pipelinerun.EventIDIn(eventIDs...)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("get pipeline runs for events: %w", err)
	}
	result := make(map[string][]PipelineRunInfo, len(runs))
	for _, r := range runs {
		info := PipelineRunInfo{
			ID:            r.ID,
			PipelineName:  r.PipelineName,
			EventID:       r.EventID,
			Status:        fmt.Sprintf("%d", r.Status),
			TriggerSource: string(r.TriggerSource),
		}
		result[r.EventID] = append(result[r.EventID], info)
	}
	return result, nil
}
