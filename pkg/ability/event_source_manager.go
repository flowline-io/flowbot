package ability

import (
	"context"
	"sync"
	"time"

	"github.com/flc1125/go-cron/v4"
	"github.com/panjf2000/ants/v2"

	"github.com/flowline-io/flowbot/pkg/metrics"
	"github.com/flowline-io/flowbot/pkg/types"
)

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

// RegisterPolling registers a polling resource with the given interval.
func (m *EventSourceManager) RegisterPolling(r PollingResource, interval time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pollers[r.ResourceName()] = &pollEntry{
		resource:    r,
		interval:    interval,
		knownHashes: make(map[string]string),
	}
}

// RegisterWebhook registers a webhook converter.
func (m *EventSourceManager) RegisterWebhook(c WebhookConverter) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.webhooks[c.WebhookPath()] = c
}

// Start begins the cron scheduler and loads persisted state.
func (m *EventSourceManager) Start(ctx context.Context) error {
	return m.startPolling(ctx)
}

// Stop shuts down the cron scheduler and flushes state.
func (m *EventSourceManager) Stop(ctx context.Context) error {
	if m.scheduler != nil {
		_ = m.scheduler.Stop()
	}
	return nil
}
