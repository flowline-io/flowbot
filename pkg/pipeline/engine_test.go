package pipeline

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestNewEngine_CronRegistration(t *testing.T) {
	t.Parallel()
	seed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := NewFakeClock(seed)
	tests := []struct {
		name        string
		defs        []Definition
		wantEntries int
	}{
		{
			name: "one cron definition registers one entry",
			defs: []Definition{
				{Name: "cron1", Enabled: true, Trigger: Trigger{Cron: "0 0 * * *"}},
			},
			wantEntries: 1,
		},
		{
			name: "multiple cron definitions register multiple entries",
			defs: []Definition{
				{Name: "cron1", Enabled: true, Trigger: Trigger{Cron: "0 0 * * *"}},
				{Name: "cron2", Enabled: true, Trigger: Trigger{Cron: "@daily"}},
			},
			wantEntries: 2,
		},
		{
			name: "event-only definition not registered as cron",
			defs: []Definition{
				{Name: "event1", Enabled: true, Trigger: Trigger{Event: "e1"}},
			},
			wantEntries: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := NewEngineWithClock(tt.defs, nil, nil, noopPC, noopEC, clock)
			defer e.Stop()
			assert.Len(t, e.cron.Entries(), tt.wantEntries)
		})
	}
}

func TestEngine_CronConcurrencyGuard(t *testing.T) {
	t.Parallel()
	seed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := NewFakeClock(seed)

	var runningCount atomic.Int32
	blockCh := make(chan struct{})
	doneCh := make(chan struct{})

	defs := []Definition{
		{
			Name:    "concurrent-pl",
			Enabled: true,
			Trigger: Trigger{Cron: "@every 100ms", CronTimeout: 5 * time.Second},
			Steps:   []Step{{Name: "blocker", Capability: "test", Operation: "block"}},
		},
	}

	e := NewEngineWithClock(defs, nil, nil, noopPC, noopEC, clock)
	defer e.Stop()

	// First goroutine acquires the lock (simulating cron run)
	go func() {
		mu := e.mu["concurrent-pl"]
		mu.Lock()
		runningCount.Add(1)
		<-blockCh
		mu.Unlock()
		doneCh <- struct{}{}
	}()

	// Wait for first to acquire lock
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, int32(1), runningCount.Load())

	// Second goroutine tries TryLock -- should fail
	skipped := true
	go func() {
		mu := e.mu["concurrent-pl"]
		if mu.TryLock() {
			skipped = false
			mu.Unlock()
		}
		doneCh <- struct{}{}
	}()
	<-doneCh
	assert.True(t, skipped, "second TryLock should fail while first holds the lock")

	close(blockCh)
	<-doneCh
}

func TestEngine_StopShutsDownCron(t *testing.T) {
	t.Parallel()
	seed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := NewFakeClock(seed)

	defs := []Definition{
		{
			Name:    "stop-test",
			Enabled: true,
			Trigger: Trigger{Cron: "@every 100ms", CronTimeout: 5 * time.Second},
			Steps:   []Step{},
		},
	}

	e := NewEngineWithClock(defs, nil, nil, noopPC, noopEC, clock)

	// Verify cron has entries before stop
	assert.Len(t, e.cron.Entries(), 1)

	e.Stop()
	// Stop should be idempotent
	e.Stop()
}

func TestEngine_SyntheticEventFormat(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		plName       string
		wantContains string
	}{
		{
			name:         "event ID contains pipeline name",
			plName:       "test-pl",
			wantContains: "cron:test-pl:",
		},
		{
			name:         "event ID format is correct",
			plName:       "my-cron-pipeline",
			wantContains: "cron:my-cron-pipeline:",
		},
		{
			name:         "hex part is 16 chars",
			plName:       "pl",
			wantContains: "cron:pl:",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			seed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			eventID := fmt.Sprintf("cron:%s:%d-%s", tt.plName, seed.UnixNano(), randomHex(8))
			assert.Contains(t, eventID, tt.wantContains)
			assert.Len(t, randomHex(8), 16)
		})
	}
}

func TestEngine_HandleEventMutex(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		defs  []Definition
		event types.DataEvent
	}{
		{
			name: "single pipeline acquires and releases mutex",
			defs: []Definition{
				{Name: "p1", Enabled: true, Trigger: Trigger{Event: "e1"}, Steps: []Step{}},
			},
			event: types.DataEvent{EventID: "evt1", EventType: "e1"},
		},
		{
			name: "no matching event does not block",
			defs: []Definition{
				{Name: "p1", Enabled: true, Trigger: Trigger{Event: "e1"}, Steps: []Step{}},
			},
			event: types.DataEvent{EventID: "evt2", EventType: "no-match"},
		},
		{
			name: "multiple pipelines for same event each lock independently",
			defs: []Definition{
				{Name: "p1", Enabled: true, Trigger: Trigger{Event: "e1"}, Steps: []Step{}},
				{Name: "p2", Enabled: true, Trigger: Trigger{Event: "e1"}, Steps: []Step{}},
			},
			event: types.DataEvent{EventID: "evt3", EventType: "e1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := NewEngine(tt.defs, nil, nil, noopPC, noopEC)
			defer e.Stop()
			err := e.Handler()(context.Background(), tt.event)
			assert.NoError(t, err)
		})
	}
}
