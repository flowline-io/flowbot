package ability

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
