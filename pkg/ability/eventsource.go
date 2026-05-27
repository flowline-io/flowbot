package ability

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"sync"
	"time"

	"github.com/flc1125/go-cron/v4"
	"github.com/gofiber/fiber/v3"
	"github.com/panjf2000/ants/v2"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/metrics"
	"github.com/flowline-io/flowbot/pkg/types"
)

// EventStore abstracts the persistence of DataEvent records.
type EventStore interface {
	AppendDataEvent(ctx context.Context, event types.DataEvent) error
	AppendEventOutbox(ctx context.Context, event types.DataEvent) error
}

// WebhookConverter converts a provider-specific webhook payload into DataEvent records.
// Each implementation encapsulates its own signature verification scheme.
type WebhookConverter interface {
	// WebhookPath returns the URL path that the webhook endpoint listens on.
	WebhookPath() string
	// VerifySignature validates the incoming webhook payload against the provider's signing scheme.
	VerifySignature(headers map[string]string, body []byte) error
	// Convert transforms a raw webhook payload into one or more DataEvent records.
	Convert(body []byte, headers map[string]string) ([]types.DataEvent, error)
}

// PollingResource represents a single pollable resource type from a provider.
// Each (provider, resource) pair registers one PollingResource.
type PollingResource interface {
	// ResourceName returns a unique name for the polled resource type.
	ResourceName() string
	// DefaultInterval returns the recommended polling interval for this resource.
	DefaultInterval() time.Duration
	// DiffKey returns a unique key from an item used to detect changes between polls.
	DiffKey(item any) string
	// ContentHash returns a hash of the item content for change detection.
	ContentHash(item any) string
	// CursorField returns the field name used for cursor-based pagination.
	CursorField() string
	// List fetches a batch of items from the provider starting after cursor.
	List(ctx context.Context, cursor string) (PollResult, error)
}

// PollResult carries a batch of items returned by a polling List call.
type PollResult struct {
	Items      []any
	NextCursor string
	HasMore    bool
}

// Persistence defines the backend storage interface for polling state.
type Persistence interface {
	LoadAll(ctx context.Context) (map[string]PollingEntry, error)
	Save(ctx context.Context, resourceName, cursor string, knownHashes map[string]string) error
}

// PollingEntry holds cursor position and known content hashes for one polling resource.
type PollingEntry struct {
	Cursor      string
	KnownHashes map[string]string
	UpdatedAt   time.Time
}

// PollingState manages in-memory polling state with periodic persistence.
// Each pollEntry has its own lock to avoid global contention.
type PollingState struct {
	mu      sync.RWMutex
	entries map[string]*pollingEntryState
	backend Persistence
	dirty   map[string]bool
}

type pollingEntryState struct {
	mu    sync.Mutex
	entry PollingEntry
}

// NewPollingState creates a PollingState backed by the given Persistence.
func NewPollingState(backend Persistence) *PollingState {
	return &PollingState{
		entries: make(map[string]*pollingEntryState),
		backend: backend,
		dirty:   make(map[string]bool),
	}
}

// Get returns a copy of the polling entry for the named resource.
// Returns an empty entry if the resource is unknown.
func (s *PollingState) Get(name string) PollingEntry {
	s.mu.RLock()
	e, ok := s.entries[name]
	s.mu.RUnlock()
	if !ok {
		return PollingEntry{KnownHashes: make(map[string]string)}
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return PollingEntry{
		Cursor:      e.entry.Cursor,
		KnownHashes: copyMap(e.entry.KnownHashes),
		UpdatedAt:   e.entry.UpdatedAt,
	}
}

// Update sets the polling entry for the named resource.
func (s *PollingState) Update(name string, entry PollingEntry) {
	s.mu.Lock()
	e, ok := s.entries[name]
	if !ok {
		e = &pollingEntryState{}
		s.entries[name] = e
	}
	s.mu.Unlock()

	e.mu.Lock()
	e.entry = PollingEntry{
		Cursor:      entry.Cursor,
		KnownHashes: copyMap(entry.KnownHashes),
		UpdatedAt:   time.Now(),
	}
	e.mu.Unlock()

	s.mu.Lock()
	s.dirty[name] = true
	s.mu.Unlock()
}

// MarkDirty marks a resource as needing persistence.
func (s *PollingState) MarkDirty(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dirty[name] = true
}

// Flush persists all dirty entries to the backend.
// It attempts to save every dirty entry and collects errors so that a single
// failure does not abandon remaining entries.
func (s *PollingState) Flush(ctx context.Context) error {
	s.mu.RLock()
	names := make([]string, 0, len(s.dirty))
	for name := range s.dirty {
		names = append(names, name)
	}
	s.mu.RUnlock()

	var errs []error
	for _, name := range names {
		s.mu.RLock()
		e, ok := s.entries[name]
		s.mu.RUnlock()
		if !ok {
			s.mu.Lock()
			delete(s.dirty, name)
			s.mu.Unlock()
			continue
		}
		e.mu.Lock()
		entry := PollingEntry{
			Cursor:      e.entry.Cursor,
			KnownHashes: copyMap(e.entry.KnownHashes),
		}
		e.mu.Unlock()

		if s.backend != nil {
			if err := s.backend.Save(ctx, name, entry.Cursor, entry.KnownHashes); err != nil {
				errs = append(errs, err)
				continue
			}
		}

		s.mu.Lock()
		delete(s.dirty, name)
		s.mu.Unlock()
	}

	if len(errs) > 0 {
		return fmt.Errorf("flush errors: %w", errors.Join(errs...))
	}
	return nil
}

// Load restores state from the persistence backend.
// It overwrites any in-memory entries with the persisted data — this is
// intended to be called once during startup before any polls run, so
// persisted state takes precedence over in-memory defaults.
func (s *PollingState) Load(ctx context.Context) error {
	if s.backend == nil {
		return nil
	}
	persisted, err := s.backend.LoadAll(ctx)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for name, pentry := range persisted {
		s.entries[name] = &pollingEntryState{
			entry: PollingEntry{
				Cursor:      pentry.Cursor,
				KnownHashes: copyMap(pentry.KnownHashes),
				UpdatedAt:   pentry.UpdatedAt,
			},
		}
	}
	return nil
}

// FlushInterval returns the recommended interval between periodic flushes.
func (*PollingState) FlushInterval() time.Duration {
	return 5 * time.Minute
}

func copyMap(src map[string]string) map[string]string {
	if src == nil {
		return make(map[string]string)
	}
	dst := make(map[string]string, len(src))
	maps.Copy(dst, src)
	return dst
}

// EventSourceEmitter is the function signature for emitting DataEvents produced
// by webhook converters and polling resources.
type EventSourceEmitter func(ctx context.Context, events []types.DataEvent) error

// EventSourceManager orchestrates webhook converters and polling resources,
// dispatching their output through the EventEmitter chain.
type EventSourceManager struct {
	mu         sync.RWMutex
	pollers    map[string]*pollEntry
	webhooks   map[string]WebhookConverter
	emitter    EventSourceEmitter
	scheduler  *cron.Cron
	stateStore *PollingState
	pool       *ants.PoolWithFunc
	metrics    *metrics.EventSourceCollector
}

// pollEntry holds the runtime state for one registered polling resource.
type pollEntry struct {
	mu                  sync.Mutex
	resource            PollingResource
	interval            time.Duration
	cursor              string
	knownHashes         map[string]string
	updatedAt           time.Time
	consecutiveFailures int
}

// NewEventSourceManager creates an EventSourceManager backed by the given emitter,
// state store, and metrics collector.
func NewEventSourceManager(
	emitter EventSourceEmitter,
	stateStore *PollingState,
	mc *metrics.EventSourceCollector,
) *EventSourceManager {
	return &EventSourceManager{
		pollers:    make(map[string]*pollEntry),
		webhooks:   make(map[string]WebhookConverter),
		emitter:    emitter,
		stateStore: stateStore,
		metrics:    mc,
	}
}

// RegisterPolling registers a polling resource. The poll interval is derived from
// the resource's DefaultInterval method.
func (m *EventSourceManager) RegisterPolling(r PollingResource) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pollers[r.ResourceName()] = &pollEntry{
		resource:    r,
		interval:    r.DefaultInterval(),
		knownHashes: make(map[string]string),
	}
}

// RegisterWebhook registers a webhook converter.
func (m *EventSourceManager) RegisterWebhook(c WebhookConverter) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.webhooks[c.WebhookPath()] = c
}

// Start begins the cron scheduler, loads persisted state, and starts periodic flush.
func (m *EventSourceManager) Start(ctx context.Context) error {
	if m.stateStore != nil {
		if err := m.stateStore.Load(ctx); err != nil {
			return fmt.Errorf("load polling state: %w", err)
		}
	}
	if err := m.startPolling(ctx); err != nil {
		return fmt.Errorf("start polling: %w", err)
	}
	m.startFlushLoop(ctx)
	return nil
}

// SetPool assigns the event pool for non-blocking webhook event submission.
func (m *EventSourceManager) SetPool(pool *ants.PoolWithFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pool = pool
}

// Stop shuts down the cron scheduler, flushes state, and releases the event pool.
func (m *EventSourceManager) Stop(ctx context.Context) error {
	if m.scheduler != nil {
		_ = m.scheduler.Stop()
	}
	if m.stateStore != nil {
		if err := m.stateStore.Flush(ctx); err != nil {
			flog.Warn("event_source: flush on stop error: %v", err)
		}
	}
	if m.pool != nil {
		m.pool.ReleaseTimeout(30 * time.Second)
	}
	return nil
}

// startFlushLoop runs a background goroutine that periodically flushes dirty state.
func (m *EventSourceManager) startFlushLoop(ctx context.Context) {
	if m.stateStore == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(m.stateStore.FlushInterval())
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				start := time.Now()
				if err := m.stateStore.Flush(context.Background()); err != nil {
					flog.Warn("event_source: periodic flush failed: %v", err)
				}
				if m.metrics != nil {
					m.metrics.ObserveStateFlushDuration(time.Since(start).Seconds())
				}
			}
		}
	}()
}

// WebhookHandler returns a Fiber handler that dispatches incoming webhook requests
// to the registered WebhookConverter for the given path.
func (m *EventSourceManager) WebhookHandler() fiber.Handler {
	return func(c fiber.Ctx) error {
		path := c.Params("*")
		if path == "" {
			return c.SendStatus(fiber.StatusNotFound)
		}

		m.mu.RLock()
		converter, ok := m.webhooks[path]
		m.mu.RUnlock()
		if !ok {
			return c.SendStatus(fiber.StatusNotFound)
		}

		body := c.Body()

		headers := make(map[string]string)
		c.Request().Header.VisitAll(func(key, value []byte) {
			headers[http.CanonicalHeaderKey(string(key))] = string(value)
		})
		c.Request().URI().QueryArgs().VisitAll(func(key, value []byte) {
			headers[http.CanonicalHeaderKey("X-Query-"+string(key))] = string(value)
		})

		if err := converter.VerifySignature(headers, body); err != nil {
			flog.Warn("event_source: webhook %s signature failed: %v", path, err)
			return c.SendStatus(fiber.StatusUnauthorized)
		}

		events, err := converter.Convert(body, headers)
		if err != nil {
			flog.Warn("event_source: webhook %s convert failed: %v", path, err)
			return c.SendStatus(fiber.StatusBadRequest)
		}

		if m.metrics != nil {
			m.metrics.IncWebhookTotal(path, "202")
			m.metrics.IncWebhookEvents(path)
		}

		for _, ev := range events {
			m.poolSubmit(func() {
				if m.emitter != nil {
					if err := m.emitter(context.Background(), []types.DataEvent{ev}); err != nil {
						flog.Error(fmt.Errorf("event_source: webhook %s emit failed: %w", path, err))
					}
				}
			})
		}

		return c.SendStatus(fiber.StatusAccepted)
	}
}

// poolSubmit submits a function to the event pool, falling back to direct execution.
func (m *EventSourceManager) poolSubmit(fn func()) {
	if m.pool != nil {
		_ = m.pool.Invoke(fn)
	} else {
		fn()
	}
}

// ----- global accessor (follows pool.go pattern) -----

var (
	globalSrcMgr   *EventSourceManager
	globalSrcMgrMu sync.Mutex
)

// SetEventSourceManager stores the EventSourceManager for cross-package access.
// Must be called during server startup before modules Bootstrap.
func SetEventSourceManager(m *EventSourceManager) {
	globalSrcMgrMu.Lock()
	defer globalSrcMgrMu.Unlock()
	globalSrcMgr = m
}

// GetEventSourceManager returns the global EventSourceManager, or nil if not set.
func GetEventSourceManager() *EventSourceManager {
	globalSrcMgrMu.Lock()
	defer globalSrcMgrMu.Unlock()
	return globalSrcMgr
}

// ----- Poll scheduler -----

const defaultPollTimeout = 30 * time.Second

// startPolling initializes the cron scheduler and registers one job per polling resource.
func (m *EventSourceManager) startPolling(_ context.Context) error {
	if len(m.pollers) == 0 {
		return nil
	}
	s := cron.New(cron.WithSeconds())
	m.scheduler = s

	for name, entry := range m.pollers {
		if m.stateStore != nil {
			storedEntry := m.stateStore.Get(name)
			if storedEntry.Cursor != "" {
				entry.cursor = storedEntry.Cursor
				entry.knownHashes = storedEntry.KnownHashes
			}
		}

		spec := fmt.Sprintf("@every %s", entry.interval.String())
		_, err := s.AddFunc(spec, func(ctx context.Context) error {
			m.pollOnce(ctx, name, entry)
			return nil
		})
		if err != nil {
			return fmt.Errorf("register cron for %s: %w", name, err)
		}
	}

	s.Start()
	return nil
}

// pollOnce executes one polling cycle for the given resource.
func (m *EventSourceManager) pollOnce(ctx context.Context, name string, entry *pollEntry) {
	timeout := max(entry.interval/2, defaultPollTimeout)
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
