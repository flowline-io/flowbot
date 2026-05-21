# Bulkhead Isolation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Isolate ability invocations by capability using semaphore-based bulkhead with bounded queue and timeout.

**Architecture:** A `Bulkhead` struct holds two channels — `sem` for concurrency tokens and `queue` for waiting slots. `Do()` acquires a queue slot (non-blocking), then waits for a semaphore token with timeout, then executes the function. A global manager creates instances lazily per capability.

**Spec:** `docs/superpowers/specs/2026-05-21-bulkhead-isolation-design.md`

**Tech Stack:** Go 1.26, stdlib channels, no third-party deps for bulkhead core.

---

## File Structure

| File | Responsibility |
|------|---------------|
| `pkg/bulkhead/bulkhead.go` | `Bulkhead` struct, `config`, `Option` pattern, `New`, `Do`, sentinel errors |
| `pkg/bulkhead/bulkhead_test.go` | TDD unit tests (9+ cases, table-driven) |
| `pkg/bulkhead/manager.go` | Global manager, `Get`, `SetDefaults` |
| `pkg/metrics/ability.go` | New bulkhead gauge/counter/histogram fields + methods |
| `pkg/ability/invoke.go` | Wrap `invoker(ctx, params)` with `bulkhead.Get(...).Do(...)` |

No files deleted. `pkg/types/errors.go` uses existing `ErrRateLimited`/`ErrTimeout` — no changes needed.

---

### Task 1: Create bulkhead types, errors, and Do method

**Files:**
- Create: `pkg/bulkhead/bulkhead.go`

- [ ] **Step 1: Create `pkg/bulkhead/` directory and `bulkhead.go`**

```go
// Package bulkhead provides per-capability semaphore isolation for ability invocations.
package bulkhead

import (
	"context"
	"errors"
	"time"
)

var (
	ErrBulkheadFull    = errors.New("bulkhead: queue full")
	ErrBulkheadTimeout = errors.New("bulkhead: wait timeout")
)

type config struct {
	maxConcurrent int
	maxQueue      int
	timeout       time.Duration
	onEnter       func(name string, waitDuration time.Duration)
	onLeave       func(name string)
	onDrop        func(name string, reason string)
}

type Option func(*config)

func WithMaxConcurrent(n int) Option {
	return func(c *config) { c.maxConcurrent = n }
}

func WithMaxQueue(n int) Option {
	return func(c *config) { c.maxQueue = n }
}

func WithTimeout(d time.Duration) Option {
	return func(c *config) { c.timeout = d }
}

func WithOnEnter(fn func(name string, waitDuration time.Duration)) Option {
	return func(c *config) { c.onEnter = fn }
}

func WithOnLeave(fn func(name string)) Option {
	return func(c *config) { c.onLeave = fn }
}

func WithOnDrop(fn func(name string, reason string)) Option {
	return func(c *config) { c.onDrop = fn }
}

// Bulkhead controls concurrent access for a named resource.
type Bulkhead struct {
	name   string
	sem    chan struct{}
	queue  chan struct{}
	config config
}

// New creates a Bulkhead with the given options.
func New(name string, opts ...Option) *Bulkhead {
	cfg := config{
		maxConcurrent: 1,
		maxQueue:      0,
		timeout:       30 * time.Second,
	}
	for _, o := range opts {
		o(&cfg)
	}
	return &Bulkhead{
		name:   name,
		sem:    make(chan struct{}, cfg.maxConcurrent),
		queue:  make(chan struct{}, cfg.maxQueue),
		config: cfg,
	}
}

// Do acquires a slot, waiting up to the configured timeout, then executes fn.
// Returns ErrBulkheadFull if the queue is at capacity.
// Returns ErrBulkheadTimeout if a slot is not acquired within the timeout.
// Returns ctx.Err() if the context is cancelled while waiting.
func (b *Bulkhead) Do(ctx context.Context, fn func() error) error {
	if b.config.maxQueue > 0 {
		select {
		case b.queue <- struct{}{}:
			defer func() { <-b.queue }()
		default:
			if b.config.onDrop != nil {
				b.config.onDrop(b.name, "queue_full")
			}
			return ErrBulkheadFull
		}
	}

	waitStart := time.Now()
	timer := time.NewTimer(b.config.timeout)
	defer timer.Stop()

	select {
	case b.sem <- struct{}{}:
	case <-timer.C:
		if b.config.onDrop != nil {
			b.config.onDrop(b.name, "timeout")
		}
		return ErrBulkheadTimeout
	case <-ctx.Done():
		if b.config.onDrop != nil {
			b.config.onDrop(b.name, "canceled")
		}
		return ctx.Err()
	}

	defer func() { <-b.sem }()

	if b.config.onEnter != nil {
		b.config.onEnter(b.name, time.Since(waitStart))
	}
	defer func() {
		if b.config.onLeave != nil {
			b.config.onLeave(b.name)
		}
	}()

	return fn()
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./pkg/bulkhead/
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add pkg/bulkhead/bulkhead.go
git commit -m "feat: add bulkhead types and Do method"
```

---

### Task 2: Write TDD unit tests for Bulkhead.Do

**Files:**
- Create: `pkg/bulkhead/bulkhead_test.go`

- [ ] **Step 1: Create `pkg/bulkhead/bulkhead_test.go`**

```go
package bulkhead

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestBulkheadDoAcquireRelease(t *testing.T) {
	tests := []struct {
		name string
		size int
	}{
		{name: "single slot", size: 1},
		{name: "two slots", size: 2},
		{name: "five slots", size: 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := New("test", WithMaxConcurrent(tt.size), WithTimeout(10*time.Second))
			var executed int32
			err := b.Do(context.Background(), func() error {
				atomic.AddInt32(&executed, 1)
				return nil
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if atomic.LoadInt32(&executed) != 1 {
				t.Fatalf("expected executed=1, got %d", executed)
			}
		})
	}
}

func TestBulkheadDoEnforcesMaxConcurrent(t *testing.T) {
	maxConc := 3
	b := New("test", WithMaxConcurrent(maxConc), WithMaxQueue(0), WithTimeout(5*time.Second))

	var current, maxConcurrent int64
	gate := make(chan struct{})
	ready := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < maxConc*3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-ready
			_ = b.Do(context.Background(), func() error {
				cur := atomic.AddInt64(&current, 1)
				for {
					old := atomic.LoadInt64(&maxConcurrent)
					if cur > old {
						if atomic.CompareAndSwapInt64(&maxConcurrent, old, cur) {
							break
						}
					} else {
						break
					}
				}
				<-gate
				atomic.AddInt64(&current, -1)
				return nil
			})
		}()
	}

	close(ready)
	time.Sleep(200 * time.Millisecond)

	if n := atomic.LoadInt64(&maxConcurrent); n != int64(maxConc) {
		t.Errorf("expected max concurrent %d, got %d", maxConc, n)
	}

	close(gate)
	wg.Wait()
}

func TestBulkheadDoContextCancellation(t *testing.T) {
	b := New("test", WithMaxConcurrent(1), WithMaxQueue(1), WithTimeout(10*time.Second))

	hold := make(chan struct{})
	go func() {
		_ = b.Do(context.Background(), func() error {
			<-hold
			return nil
		})
	}()

	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- b.Do(ctx, func() error { return nil })
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for cancellation")
	}

	close(hold)
}

func TestBulkheadDoTimeout(t *testing.T) {
	b := New("test", WithMaxConcurrent(1), WithMaxQueue(1), WithTimeout(50*time.Millisecond))

	hold := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = b.Do(context.Background(), func() error {
			<-hold
			return nil
		})
	}()

	time.Sleep(100 * time.Millisecond)

	err := b.Do(context.Background(), func() error { return nil })
	if !errors.Is(err, ErrBulkheadTimeout) {
		t.Errorf("expected ErrBulkheadTimeout, got %v", err)
	}

	close(hold)
	<-done
}

func TestBulkheadDoQueueFull(t *testing.T) {
	b := New("test", WithMaxConcurrent(1), WithMaxQueue(1), WithTimeout(5*time.Second))

	hold := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = b.Do(context.Background(), func() error {
			<-hold
			return nil
		})
	}()

	time.Sleep(50 * time.Millisecond)

	queued := make(chan struct{})
	go func() {
		close(queued)
		_ = b.Do(context.Background(), func() error { return nil })
	}()
	<-queued
	time.Sleep(50 * time.Millisecond)

	err := b.Do(context.Background(), func() error { return nil })
	if !errors.Is(err, ErrBulkheadFull) {
		t.Errorf("expected ErrBulkheadFull, got %v", err)
	}

	close(hold)
	<-done
}

func TestBulkheadDoCallbacks(t *testing.T) {
	tests := []struct {
		name           string
		trigger        func(b *Bulkhead)
		wantEnterCalls int32
		wantLeaveCalls int32
		wantDropCalls  int32
		wantDropReason string
	}{
		{
			name: "successful call triggers enter and leave",
			trigger: func(b *Bulkhead) {
				_ = b.Do(context.Background(), func() error { return nil })
			},
			wantEnterCalls: 1,
			wantLeaveCalls: 1,
			wantDropCalls:  0,
		},
		{
			name: "timeout triggers drop",
			trigger: func(b *Bulkhead) {
				hold := make(chan struct{})
				go func() {
					_ = b.Do(context.Background(), func() error { <-hold; return nil })
				}()
				time.Sleep(100 * time.Millisecond)
				err := b.Do(context.Background(), func() error { return nil })
				if !errors.Is(err, ErrBulkheadTimeout) {
					t.Errorf("expected ErrBulkheadTimeout, got %v", err)
				}
				close(hold)
			},
			wantEnterCalls: 1,
			wantLeaveCalls: 1,
			wantDropCalls:  1,
			wantDropReason: "timeout",
		},
		{
			name: "queue full triggers drop",
			trigger: func(b *Bulkhead) {
				hold := make(chan struct{})
				go func() {
					_ = b.Do(context.Background(), func() error { <-hold; return nil })
				}()
				time.Sleep(50 * time.Millisecond)
				go func() {
					_ = b.Do(context.Background(), func() error { return nil })
				}()
				time.Sleep(50 * time.Millisecond)
				_ = b.Do(context.Background(), func() error { return nil })
				close(hold)
			},
			wantEnterCalls: 1,
			wantLeaveCalls: 1,
			wantDropCalls:  1,
			wantDropReason: "queue_full",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var enters, leaves, drops int32
			var dropReason string
			b := New("test",
				WithMaxConcurrent(1),
				WithMaxQueue(1),
				WithTimeout(50*time.Millisecond),
				WithOnEnter(func(name string, d time.Duration) {
					atomic.AddInt32(&enters, 1)
				}),
				WithOnLeave(func(name string) {
					atomic.AddInt32(&leaves, 1)
				}),
				WithOnDrop(func(name string, reason string) {
					atomic.AddInt32(&drops, 1)
					dropReason = reason
				}),
			)
			tt.trigger(b)
			if atomic.LoadInt32(&enters) != tt.wantEnterCalls {
				t.Errorf("enters: want %d, got %d", tt.wantEnterCalls, enters)
			}
			if atomic.LoadInt32(&leaves) != tt.wantLeaveCalls {
				t.Errorf("leaves: want %d, got %d", tt.wantLeaveCalls, leaves)
			}
			if atomic.LoadInt32(&drops) != tt.wantDropCalls {
				t.Errorf("drops: want %d, got %d", tt.wantDropCalls, drops)
			}
			if tt.wantDropCalls > 0 && dropReason != tt.wantDropReason {
				t.Errorf("drop reason: want %s, got %s", tt.wantDropReason, dropReason)
			}
		})
	}
}

func TestBulkheadDefaultSize(t *testing.T) {
	n := runtime.GOMAXPROCS(0)
	if n < 1 {
		t.Fatal("GOMAXPROCS returned 0")
	}
	expected := n * 4
	b := New("test", WithMaxConcurrent(expected), WithMaxQueue(expected))
	if cap(b.sem) != expected {
		t.Errorf("sem cap: want %d, got %d", expected, cap(b.sem))
	}
	if cap(b.queue) != expected {
		t.Errorf("queue cap: want %d, got %d", expected, cap(b.queue))
	}
}

func TestBulkheadDoRace(t *testing.T) {
	b := New("test", WithMaxConcurrent(4), WithMaxQueue(4), WithTimeout(5*time.Second))
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = b.Do(context.Background(), func() error { return nil })
		}()
	}
	wg.Wait()
}
```

- [ ] **Step 2: Run tests**

```bash
go test -race -count=1 ./pkg/bulkhead/ -v
```

Expected: all tests pass, no race conditions.

- [ ] **Step 3: Commit**

```bash
git add pkg/bulkhead/bulkhead_test.go
git commit -m "test: add TDD tests for bulkhead Do method"
```

---

### Task 3: Create global manager with lazy Get

**Files:**
- Create: `pkg/bulkhead/manager.go`

- [ ] **Step 1: Create `pkg/bulkhead/manager.go`**

```go
package bulkhead

import (
	"runtime"
	"sync"
	"time"
)

type manager struct {
	mu        sync.Mutex
	instances map[string]*Bulkhead
	defaults  config
}

var defaultManager = &manager{instances: make(map[string]*Bulkhead)}

func init() {
	n := runtime.GOMAXPROCS(0)
	defaultManager.defaults = config{
		maxConcurrent: n * 4,
		maxQueue:      n * 4,
		timeout:       30 * time.Second,
	}
}

// Get returns the Bulkhead for the given name, creating one with default config if needed.
func Get(name string) *Bulkhead {
	defaultManager.mu.Lock()
	defer defaultManager.mu.Unlock()

	if b, ok := defaultManager.instances[name]; ok {
		return b
	}

	cfg := defaultManager.defaults
	b := New(name,
		WithMaxConcurrent(cfg.maxConcurrent),
		WithMaxQueue(cfg.maxQueue),
		WithTimeout(cfg.timeout),
		WithOnEnter(cfg.onEnter),
		WithOnLeave(cfg.onLeave),
		WithOnDrop(cfg.onDrop),
	)
	defaultManager.instances[name] = b
	return b
}

// SetDefaults sets default options applied to all Bulkhead instances created via Get.
// Must be called before any Get calls for the settings to take effect.
func SetDefaults(opts ...Option) {
	defaultManager.mu.Lock()
	defer defaultManager.mu.Unlock()
	for _, o := range opts {
		o(&defaultManager.defaults)
	}
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./pkg/bulkhead/
```

- [ ] **Step 3: Commit**

```bash
git add pkg/bulkhead/manager.go
git commit -m "feat: add bulkhead manager with lazy Get"
```

---

### Task 4: Write tests for manager

**Files:**
- Create: `pkg/bulkhead/manager_test.go`

- [ ] **Step 1: Create `pkg/bulkhead/manager_test.go`**

```go
package bulkhead

import (
	"runtime"
	"testing"
)

func TestGetReturnsSingleton(t *testing.T) {
	a := Get("foo")
	b := Get("foo")
	if a != b {
		t.Error("Get should return the same instance for the same name")
	}
}

func TestGetDifferentNamesReturnDifferentInstances(t *testing.T) {
	a := Get("bar")
	b := Get("baz")
	if a == b {
		t.Error("Get should return different instances for different names")
	}
}

func TestGetDefaultConfig(t *testing.T) {
	b := Get("default-test")
	n := runtime.GOMAXPROCS(0)
	expected := n * 4
	if cap(b.sem) != expected {
		t.Errorf("default sem cap: want %d, got %d", expected, cap(b.sem))
	}
	if cap(b.queue) != expected {
		t.Errorf("default queue cap: want %d, got %d", expected, cap(b.queue))
	}
	if b.config.timeout.String() != "30s" {
		t.Errorf("default timeout: want 30s, got %s", b.config.timeout)
	}
}

func TestSetDefaultsAppliesToNewInstances(t *testing.T) {
	SetDefaults(
		WithMaxConcurrent(5),
		WithMaxQueue(3),
	)
	b := Get("set-defaults-test")
	if cap(b.sem) != 5 {
		t.Errorf("sem cap: want 5, got %d", cap(b.sem))
	}
	if cap(b.queue) != 3 {
		t.Errorf("queue cap: want 3, got %d", cap(b.queue))
	}
}
```

- [ ] **Step 2: Run manager tests only**

```bash
go test -race -count=1 ./pkg/bulkhead/ -run TestGet -v
```

Expected: all pass.

- [ ] **Step 3: Run all bulkhead tests**

```bash
go test -race -count=1 ./pkg/bulkhead/ -v
```

Expected: all tests pass, -race clean.

- [ ] **Step 4: Commit**

```bash
git add pkg/bulkhead/manager_test.go
git commit -m "test: add manager singleton tests"
```

---

### Task 5: Add bulkhead metrics to AbilityCollector

**Files:**
- Modify: `pkg/metrics/ability.go`

- [ ] **Step 1: Add bulkhead metric fields and registration to `pkg/metrics/ability.go`**

In `AbilityCollector` struct (after `eventDroppedTotal`):

```go
bulkheadQueued     *prometheus.GaugeVec
bulkheadActive     *prometheus.GaugeVec
bulkheadDroppedTotal *prometheus.CounterVec
bulkheadWaitDuration  *prometheus.HistogramVec
```

In `NewAbilityCollector` (after the `eventDroppedTotal` registration block):

```go
c.bulkheadQueued, err = st.RegisterGaugeVec("ability_bulkhead_queued", "Invocations queued in bulkhead by capability", "capability")
if err != nil {
    log.Printf("[metrics] ability: failed to register bulkhead_queued gauge: %v", err)
    return &AbilityCollector{}
}
c.bulkheadActive, err = st.RegisterGaugeVec("ability_bulkhead_active", "Invocations active in bulkhead by capability", "capability")
if err != nil {
    log.Printf("[metrics] ability: failed to register bulkhead_active gauge: %v", err)
    return &AbilityCollector{}
}
c.bulkheadDroppedTotal, err = st.RegisterCounterVec("ability_bulkhead_dropped_total", "Invocations dropped by bulkhead by capability and reason", "capability", "reason")
if err != nil {
    log.Printf("[metrics] ability: failed to register bulkhead_dropped counter: %v", err)
    return &AbilityCollector{}
}
c.bulkheadWaitDuration, err = st.RegisterHistogramVec("ability_bulkhead_wait_seconds", "Bulkhead queue wait duration by capability", "capability")
if err != nil {
    log.Printf("[metrics] ability: failed to register bulkhead_wait histogram: %v", err)
    return &AbilityCollector{}
}
```

At end of file, after existing methods:

```go
func (c *AbilityCollector) IncBulkheadQueued(capability string) {
	if c.bulkheadQueued == nil {
		return
	}
	defer recoverLog("ability_bulkhead_queued")
	c.bulkheadQueued.WithLabelValues(sanitizeLabel(capability)).Inc()
}

func (c *AbilityCollector) DecBulkheadQueued(capability string) {
	if c.bulkheadQueued == nil {
		return
	}
	defer recoverLog("ability_bulkhead_queued")
	c.bulkheadQueued.WithLabelValues(sanitizeLabel(capability)).Dec()
}

func (c *AbilityCollector) IncBulkheadActive(capability string) {
	if c.bulkheadActive == nil {
		return
	}
	defer recoverLog("ability_bulkhead_active")
	c.bulkheadActive.WithLabelValues(sanitizeLabel(capability)).Inc()
}

func (c *AbilityCollector) DecBulkheadActive(capability string) {
	if c.bulkheadActive == nil {
		return
	}
	defer recoverLog("ability_bulkhead_active")
	c.bulkheadActive.WithLabelValues(sanitizeLabel(capability)).Dec()
}

func (c *AbilityCollector) IncBulkheadDropped(capability string, reason string) {
	if c.bulkheadDroppedTotal == nil {
		return
	}
	defer recoverLog("ability_bulkhead_dropped_total")
	c.bulkheadDroppedTotal.WithLabelValues(sanitizeLabel(capability), sanitizeLabel(reason)).Inc()
}

func (c *AbilityCollector) ObserveBulkheadWaitDuration(capability string, seconds float64) {
	if c.bulkheadWaitDuration == nil {
		return
	}
	defer recoverLog("ability_bulkhead_wait_seconds")
	c.bulkheadWaitDuration.WithLabelValues(sanitizeLabel(capability)).Observe(seconds)
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./pkg/metrics/
```

Expected: no errors.

- [ ] **Step 3: Update the noop test to cover new methods**

In `pkg/metrics/ability_test.go`, add to the `tests` slice in `TestAbilityCollector_NoopMethodsDontPanic`:

```go
{name: "IncBulkheadQueued", fn: func() { c.IncBulkheadQueued("c") }},
{name: "DecBulkheadQueued", fn: func() { c.DecBulkheadQueued("c") }},
{name: "IncBulkheadActive", fn: func() { c.IncBulkheadActive("c") }},
{name: "DecBulkheadActive", fn: func() { c.DecBulkheadActive("c") }},
{name: "IncBulkheadDropped", fn: func() { c.IncBulkheadDropped("c", "timeout") }},
{name: "ObserveBulkheadWaitDuration", fn: func() { c.ObserveBulkheadWaitDuration("c", 0.5) }},
```

- [ ] **Step 4: Run metrics tests**

```bash
go test -race -count=1 ./pkg/metrics/ -v
```

Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add pkg/metrics/ability.go pkg/metrics/ability_test.go
git commit -m "feat: add bulkhead metrics to AbilityCollector"
```

---

### Task 6: Integrate bulkhead into ability.Invoke

**Files:**
- Modify: `pkg/ability/invoke.go`

- [ ] **Step 1: Wire bulkhead callbacks in ability package**

Add to `pkg/ability/invoke.go`:

```go
import (
	// ... existing imports ...
	"github.com/flowline-io/flowbot/pkg/bulkhead"
)
```

After `SetMetricsCollector` (line 49), add:

```go
// SetBulkheadCallbacks wires the bulkhead manager with metrics reporting callbacks.
func SetBulkheadCallbacks() {
    bulkhead.SetDefaults(
        bulkhead.WithOnEnter(func(name string, d time.Duration) {
            DefaultRegistry.mu.RLock()
            mc := DefaultRegistry.metrics
            DefaultRegistry.mu.RUnlock()
            if mc != nil {
                mc.IncBulkheadActive(name)
                mc.ObserveBulkheadWaitDuration(name, d.Seconds())
            }
        }),
        bulkhead.WithOnLeave(func(name string) {
            DefaultRegistry.mu.RLock()
            mc := DefaultRegistry.metrics
            DefaultRegistry.mu.RUnlock()
            if mc != nil {
                mc.DecBulkheadActive(name)
            }
        }),
        bulkhead.WithOnDrop(func(name string, reason string) {
            DefaultRegistry.mu.RLock()
            mc := DefaultRegistry.metrics
            DefaultRegistry.mu.RUnlock()
            if mc != nil {
                mc.IncBulkheadDropped(name, reason)
            }
        }),
    )
}
```

- [ ] **Step 2: Wrap invoker call with bulkhead**

In `Invoke` method, replace lines 213-214:

```go
	start := time.Now()
	result, err := invoker(ctx, params)
```

With:

```go
	start := time.Now()
	var result *InvokeResult
	invokeErr := bulkhead.Get(string(capability)).Do(ctx, func() error {
		var err error
		result, err = invoker(ctx, params)
		return err
	})
	if invokeErr != nil {
		if errors.Is(invokeErr, bulkhead.ErrBulkheadFull) {
			err = types.Errorf(types.ErrRateLimited, "bulkhead full for %s: %v", capability, invokeErr)
		} else if errors.Is(invokeErr, bulkhead.ErrBulkheadTimeout) {
			err = types.Errorf(types.ErrTimeout, "bulkhead timeout for %s: %v", capability, invokeErr)
		} else {
			err = invokeErr
		}
		trace.RecordError(ctx, err)
		r.recordErrorMetrics(capability, operation, start, err)
		return nil, err
	}
```

Add `errors` to imports:

```go
import (
	"errors"
	// ... existing stdlib imports ...
)
```

- [ ] **Step 3: Call SetBulkheadCallbacks during startup**

In `internal/server/pipeline.go:51`, add `ability.SetBulkheadCallbacks()` right after `ability.SetMetricsCollector(ac)`:

```go
ability.SetMetricsCollector(ac)
ability.SetBulkheadCallbacks()
```

- [ ] **Step 4: Verify compilation**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 5: Run existing ability tests**

```bash
go test -race -count=1 ./pkg/ability/ -v
```

Expected: existing tests still pass.

- [ ] **Step 6: Commit**

```bash
git add pkg/ability/invoke.go internal/server/pipeline.go
git commit -m "feat: integrate bulkhead isolation into ability.Invoke"
```

---

### Task 7: Final verification

- [ ] **Step 1: Run all bulkhead tests**

```bash
go test -race -count=1 ./pkg/bulkhead/ -v
```

- [ ] **Step 2: Run all ability tests**

```bash
go test -race -count=1 ./pkg/ability/ -v
```

- [ ] **Step 3: Run all metrics tests**

```bash
go test -race -count=1 ./pkg/metrics/ -v
```

- [ ] **Step 4: Run full test suite**

```bash
go tool task test
```

- [ ] **Step 5: Run lint**

```bash
go tool task lint
```

- [ ] **Step 6: If all passes, commit any remaining changes and provide summary**
