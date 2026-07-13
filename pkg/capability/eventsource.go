package capability

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/flc1125/go-cron/v4"
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
// the resource's DefaultInterval method. Nil resources are silently skipped.
func (m *EventSourceManager) RegisterPolling(r PollingResource) {
	if r == nil {
		return
	}
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
