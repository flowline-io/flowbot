package ability

import (
	"context"
	"fmt"
	"time"

	"github.com/flc1125/go-cron/v4"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

const defaultPollTimeout = 30 * time.Second

// startPolling initializes the cron scheduler and registers one job per polling resource.
func (m *EventSourceManager) startPolling(ctx context.Context) error {
	if len(m.pollers) == 0 {
		return nil
	}
	s := cron.New(cron.WithSeconds())
	m.scheduler = s

	for name, entry := range m.pollers {
		name := name
		entry := entry

		if m.stateStore != nil {
			storedEntry := m.stateStore.Get(name)
			if storedEntry.Cursor != "" {
				entry.cursor = storedEntry.Cursor
				entry.knownHashes = storedEntry.KnownHashes
			}
		}

		interval := entry.resource.DefaultInterval()
		spec := fmt.Sprintf("@every %s", interval.String())
		_, err := s.AddFunc(spec, func(ctx context.Context) error {
			m.pollOnce(ctx, name, entry)
			return nil
		})
		if err != nil {
			return fmt.Errorf("register cron for %s: %w", name, err)
		}
		flog.Info("event_source: polling registered %s interval=%s", name, interval)
	}

	s.Start()
	return nil
}

// pollOnce executes one polling cycle for the given resource.
func (m *EventSourceManager) pollOnce(ctx context.Context, name string, entry *pollEntry) {
	timeout := entry.interval / 2
	if timeout < defaultPollTimeout {
		timeout = defaultPollTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	entry.mu.Lock()
	cursor := entry.cursor
	entry.mu.Unlock()

	result, err := entry.resource.List(ctx, cursor)
	if err != nil {
		if m.metrics != nil {
			m.metrics.IncPollError(name)
		}
		entry.mu.Lock()
		entry.consecutiveFailures++
		failures := entry.consecutiveFailures
		entry.mu.Unlock()
		if failures >= 3 {
			flog.Warn("event_source: %s polling failing repeatedly (%d failures): %v", name, failures, err)
		}
		return
	}

	if m.metrics != nil {
		m.metrics.ObservePollDuration(name, time.Since(start).Seconds())
		m.metrics.IncPollTotal(name, "success")
	}

	entry.mu.Lock()
	entry.consecutiveFailures = 0
	entry.mu.Unlock()

	newEvents := m.diffAndEmit(ctx, entry, result.Items)

	entry.mu.Lock()
	if result.NextCursor != "" {
		entry.cursor = result.NextCursor
	}
	entry.knownHashes = buildHashSet(result.Items, entry.resource.DiffKey, entry.resource.ContentHash)
	entry.updatedAt = time.Now()
	entry.mu.Unlock()

	if m.stateStore != nil {
		entry.mu.Lock()
		cursor := entry.cursor
		hashes := copyMap(entry.knownHashes)
		entry.mu.Unlock()
		m.stateStore.Update(name, PollingEntry{
			Cursor:      cursor,
			KnownHashes: hashes,
			UpdatedAt:   time.Now(),
		})
		m.stateStore.MarkDirty(name)
	}

	if m.metrics != nil {
		for _, ev := range newEvents {
			m.metrics.IncPollEvents(name, ev.EventType)
		}
	}
}

// diffAndEmit compares items against the entry's known hashes and emits
// created or updated events for differences.
func (m *EventSourceManager) diffAndEmit(ctx context.Context, entry *pollEntry, items []any) []types.DataEvent {
	var newEvents []types.DataEvent

	entry.mu.Lock()
	defer entry.mu.Unlock()

	for _, item := range items {
		key := entry.resource.DiffKey(item)
		newHash := entry.resource.ContentHash(item)
		oldHash, exists := entry.knownHashes[key]

		var eventType string
		switch {
		case !exists:
			eventType = entry.resource.ResourceName() + ".created"
		case exists && oldHash != newHash:
			eventType = entry.resource.ResourceName() + ".updated"
		default:
			continue
		}

		ev := types.DataEvent{
			EventID:        types.Id(),
			EventType:      eventType,
			Source:         "provider_event",
			IdempotencyKey: key,
			CreatedAt:      time.Now(),
			Data:           types.KV{"item": item},
		}
		newEvents = append(newEvents, ev)
	}

	if len(newEvents) > 0 && m.emitter != nil {
		_ = m.emitter(ctx, newEvents)
	}

	return newEvents
}

// buildHashSet constructs a map of DiffKey → ContentHash from a batch of items.
func buildHashSet(items []any, diffKeyFn func(any) string, contentHashFn func(any) string) map[string]string {
	hashes := make(map[string]string, len(items))
	for _, item := range items {
		hashes[diffKeyFn(item)] = contentHashFn(item)
	}
	return hashes
}
