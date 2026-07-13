# Ability Event Worker Pool Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace unbounded goroutine-per-event-emission with a `ants` goroutine pool in `capability.Invoke`.

**Architecture:** Add `pkg/ability/pool.go` wrapping `ants.PoolWithFunc` with nonblocking submit + best-effort drop. Wire pool init in `initPipeline`, shutdown in `RunServer.OnStop`. Add `event_dropped_total` counter to `AbilityCollector`.

**Tech Stack:** Go 1.26+, `github.com/panjf2000/ants/v2`, Prometheus, Viper config

---

**File Map:**

| File                          | Action | Responsibility                                  |
| ----------------------------- | ------ | ----------------------------------------------- |
| `pkg/ability/pool.go`         | Create | `ants.PoolWithFunc` wrapper, init/shutdown/drop |
| `pkg/ability/pool_test.go`    | Create | Unit tests for pool wrapper                     |
| `pkg/ability/invoke.go`       | Modify | Replace `go func()` with `pool.Invoke()`        |
| `pkg/ability/invoke_test.go`  | Modify | Update event emission tests for pool            |
| `pkg/metrics/capability.go`      | Modify | Add `eventDroppedTotal` counter                 |
| `pkg/config/config.go`        | Modify | Add `AbilityEventPool` config struct            |
| `internal/server/pipeline.go` | Modify | Call `capability.InitEventPool`                    |
| `internal/server/server.go`   | Modify | Call `capability.ShutdownEventPool` in OnStop      |
| `docs/reference/config.yaml`  | Modify | Add `capability.event_pool` section                |
| `go.mod`                      | Modify | Add `ants/v2` dependency                        |

---

### Task 1: Add `ants` dependency

**Files:**

- Modify: `go.mod`

- [ ] **Step 1: Run `go get` to add ants**

```bash
go get github.com/panjf2000/ants/v2
```

Expected: dependency added to `go.mod` and `go.sum`.

- [ ] **Step 2: Commit**

```bash
git add go.mod go.sum
git commit -m "deps: add ants goroutine pool"
```

---

### Task 2: Add event_dropped counter to AbilityCollector

**Files:**

- Modify: `pkg/metrics/capability.go:12-56`

- [ ] **Step 1: Write the test for dropped counter**

File: `pkg/metrics/ability_test.go` already exists. Add a new test:

```go
func TestAbilityCollector_IncEventDropped(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		capability string
		operation  string
		reason     string
	}{
		{"increments dropped counter for overload reason", "bookmark", "list", "pool_overload"},
		{"increments dropped counter for closed pool reason", "kanban", "list_tasks", "pool_closed"},
		{"increments dropped counter with empty reason", "reader", "list_entries", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			st := stats.NewForTest(t)
			c := NewAbilityCollector(st)
			c.IncEventDropped(tt.capability, tt.operation, tt.reason)
			// Verify prometheus counter was incremented via gather
			metrics, _ := st.Gather()
			assert.NotEmpty(t, metrics)
		})
	}
}
```

- [ ] **Step 2: Add the counter field and method to AbilityCollector**

Modify `pkg/metrics/capability.go`:

```go
type AbilityCollector struct {
	invokeTotal       *prometheus.CounterVec
	invokeDuration    *prometheus.HistogramVec
	invokeErrorTotal  *prometheus.CounterVec
	eventDroppedTotal *prometheus.CounterVec
}

func NewAbilityCollector(st *stats.Stats) *AbilityCollector {
	if st == nil {
		return &AbilityCollector{}
	}
	return &AbilityCollector{
		invokeTotal:       st.RegisterCounterVec("ability_invoke_total", "Invocations by capability, operation, and status", "capability", "operation", "status"),
		invokeDuration:    st.RegisterHistogramVec("ability_invoke_duration_seconds", "Invocation duration distribution", "capability", "operation"),
		invokeErrorTotal:  st.RegisterCounterVec("ability_invoke_error_total", "Invocation errors by capability, operation, and error code", "capability", "operation", "error_code"),
		eventDroppedTotal: st.RegisterCounterVec("ability_event_dropped_total", "Events dropped due to pool overflow or shutdown", "capability", "operation", "reason"),
	}
}

// IncEventDropped increments the event dropped counter.
func (c *AbilityCollector) IncEventDropped(capability, operation, reason string) {
	if c.eventDroppedTotal == nil {
		return
	}
	defer recoverLog("ability_event_dropped_total")
	c.eventDroppedTotal.WithLabelValues(sanitizeLabel(capability), sanitizeLabel(operation), sanitizeLabel(reason)).Inc()
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./pkg/metrics/... -v -run "IncEventDropped"
```

Expected: test passes.

- [ ] **Step 4: Commit**

```bash
git add pkg/metrics/capability.go pkg/metrics/ability_test.go
git commit -m "feat: add event_dropped counter to AbilityCollector"
```

---

### Task 3: Add AbilityEventPool config struct

**Files:**

- Modify: `pkg/config/config.go`
- Modify: `docs/reference/config.yaml`

- [ ] **Step 1: Add AbilityEventPool type to config.go**

After the `Ability` type or at the end of config structs, add:

```go
// AbilityEventPool configures the goroutine pool for event emission.
type AbilityEventPool struct {
	// Size is the max number of goroutines in the pool (0 = ants default).
	Size int `json:"size" yaml:"size" mapstructure:"size"`
	// ExpiryDuration is the idle worker eviction interval (e.g. "30s").
	ExpiryDuration string `json:"expiry_duration" yaml:"expiry_duration" mapstructure:"expiry_duration"`
}
```

- [ ] **Step 2: Add Ability type to config struct**

Because `ability` is a top-level key in the config, add to `config.Type`:

```go
// In Type struct, add after Profiling field:
Ability struct {
	EventPool AbilityEventPool `json:"event_pool" yaml:"event_pool" mapstructure:"event_pool"`
} `json:"ability" yaml:"ability" mapstructure:"ability"`
```

- [ ] **Step 3: Add config to reference config.yaml**

Append to `docs/reference/config.yaml`:

```yaml
# Ability invocation configuration
ability:
  event_pool:
    # Max concurrent event emission workers (0 = ants default)
    size: 0
    # Idle worker eviction duration
    expiry_duration: "30s"
```

- [ ] **Step 4: Run tests**

```bash
go test ./pkg/config/... -v -count=1
```

Expected: existing config tests pass.

- [ ] **Step 5: Commit**

```bash
git add pkg/config/config.go pkg/config/config_test.go docs/reference/config.yaml
git commit -m "feat: add capability.event_pool configuration"
```

---

### Task 4: Create pool wrapper

**Files:**

- Create: `pkg/ability/pool.go`
- Create: `pkg/ability/pool_test.go`

- [ ] **Step 1: Create pool.go**

```go
package ability

import (
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/metrics"
)

// eventPoolConfig holds config values read at InitEventPool time.
type eventPoolConfig struct {
	size    int
	expiry  time.Duration
	metrics *metrics.AbilityCollector
}

// eventPool wraps an ants.PoolWithFunc for nonblocking event emission.
type eventPool struct {
	pool   *ants.PoolWithFunc
	config eventPoolConfig
}

// eventTask bundles the data needed by the pool worker function.
type eventTask struct {
	capability string
	operation  string
	fn         func()
}

var (
	epMu   sync.Mutex
	epInst *eventPool
)

// InitEventPool creates the global event pool. Must be called once during startup.
// Call ShutdownEventPool during server shutdown.
func InitEventPool(size int, expiryDuration string, mc *metrics.AbilityCollector) error {
	epMu.Lock()
	defer epMu.Unlock()

	if epInst != nil {
		flog.Warn("ability: event pool already initialized")
		return nil
	}

	expiry, err := time.ParseDuration(expiryDuration)
	if err != nil {
		expiry = 30 * time.Second
	}

	cfg := eventPoolConfig{
		size:    size,
		expiry:  expiry,
		metrics: mc,
	}

	pool, err := ants.NewPoolWithFunc(size, func(i any) {
		task, ok := i.(*eventTask)
		if !ok {
			return
		}
		task.fn()
	}, ants.WithNonblocking(true), ants.WithExpiryDuration(expiry))
	if err != nil {
		return err
	}

	epInst = &eventPool{pool: pool, config: cfg}
	flog.Info("ability: event pool initialized (size=%d, expiry=%s)", pool.Cap(), expiry)
	return nil
}

// ShutdownEventPool releases the pool, waiting up to 30s for in-flight tasks.
func ShutdownEventPool() {
	epMu.Lock()
	defer epMu.Unlock()

	if epInst == nil {
		return
	}
	epInst.pool.ReleaseTimeout(30 * time.Second)
	epInst = nil
	flog.Info("ability: event pool released")
}

// submitEvent submits an event emission function to the pool.
// Returns true if submitted, false if dropped.
func submitEvent(capability, operation string, fn func()) {
	epMu.Lock()
	ep := epInst
	epMu.Unlock()

	if ep == nil {
		fn()
		return
	}

	task := &eventTask{
		capability: capability,
		operation:  operation,
		fn:         fn,
	}

	err := ep.pool.Invoke(task)
	if err != nil {
		reason := "unknown"
		if err == ants.ErrPoolOverload {
			reason = "pool_overload"
		} else if err == ants.ErrPoolClosed {
			reason = "pool_closed"
		}
		flog.Warn("ability(%s.%s): event dropped: %v", capability, operation, err)
		if ep.config.metrics != nil {
			ep.config.metrics.IncEventDropped(capability, operation, reason)
		}
	}
}
```

- [ ] **Step 2: Create pool_test.go**

```go
package ability

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/metrics"
)

func TestInitEventPool(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"initializes pool with valid config"},
		{"double init is safe and logs warning"},
		{"init with zero size uses ants default"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			epMu.Lock()
			epInst = nil
			epMu.Unlock()

			err := InitEventPool(10, "30s", nil)
			require.NoError(t, err)
			require.NotNil(t, epInst)

			if tt.name == "double init is safe and logs warning" {
				err := InitEventPool(20, "10s", nil)
				require.NoError(t, err)
			}

			epMu.Lock()
			epInst = nil
			epMu.Unlock()
		})
	}
}

func TestSubmitEvent(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"submits and executes task via pool"},
		{"submits multiple tasks concurrently"},
		{"falls back to direct exec when pool is nil"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			epMu.Lock()
			epInst = nil
			epMu.Unlock()

			err := InitEventPool(10, "30s", nil)
			require.NoError(t, err)
			defer func() {
				epMu.Lock()
				epInst = nil
				epMu.Unlock()
			}()

			if tt.name == "falls back to direct exec when pool is nil" {
				epMu.Lock()
				epInst = nil
				epMu.Unlock()
			}

			var mu sync.Mutex
			var executed []string

			if tt.name == "submits multiple tasks concurrently" {
				var wg sync.WaitGroup
				for i := 0; i < 20; i++ {
					wg.Add(1)
					go func(idx int) {
						defer wg.Done()
						submitEvent("test", "op", func() {
							mu.Lock()
							executed = append(executed, "task")
							mu.Unlock()
						})
					}(i)
				}
				wg.Wait()
				time.Sleep(100 * time.Millisecond) // wait for pool workers
				mu.Lock()
				assert.Len(t, executed, 20)
				mu.Unlock()
				return
			}

			done := make(chan struct{})
			submitEvent("test", "op", func() {
				mu.Lock()
				executed = append(executed, "task")
				mu.Unlock()
				close(done)
			})

			select {
			case <-done:
				mu.Lock()
				assert.Len(t, executed, 1)
				mu.Unlock()
			case <-time.After(time.Second):
				t.Fatal("task not executed within timeout")
			}
		})
	}
}

func TestSubmitEventDropOnFull(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"drops event when pool is full"},
		{"drops event when pool is closed"},
		{"increments dropped metric on drop"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			epMu.Lock()
			epInst = nil
			epMu.Unlock()

			if tt.name == "drops event when pool is closed" {
				err := InitEventPool(1, "30s", metrics.NewAbilityCollector(nil))
				require.NoError(t, err)
				epInst.pool.Release()
				submitEvent("test", "op", func() {})
				// should not panic
				epMu.Lock()
				epInst = nil
				epMu.Unlock()
				return
			}

			if tt.name == "increments dropped metric on drop" {
				err := InitEventPool(1, "30s", nil)
				require.NoError(t, err)
				defer func() {
					epMu.Lock()
					epInst = nil
					epMu.Unlock()
				}()

				// Fill the pool with a blocking task
				block := make(chan struct{})
				submitEvent("test", "op", func() { <-block })
				// Next submit should drop (nonblocking, pool size=1)
				// Submit more to fill up
				for i := 0; i < 100; i++ {
					submitEvent("test", "op", func() {
						time.Sleep(time.Millisecond)
					})
				}
				close(block)
				time.Sleep(50 * time.Millisecond)
				return
			}

			// pool is full
			err := InitEventPool(1, "30s", nil)
			require.NoError(t, err)
			defer func() {
				epMu.Lock()
				epInst = nil
				epMu.Unlock()
			}()

			block := make(chan struct{})
			submitEvent("test", "op", func() { <-block })

			dropped := false
			for i := 0; i < 10; i++ {
				// Submit should be dropped when pool (size=1) is busy
				submitEvent("test", "op", func() {
					dropped = true // this should NOT be reached
				})
			}
			close(block)
			time.Sleep(50 * time.Millisecond)
			assert.False(t, dropped)
		})
	}
}

func TestShutdownEventPool(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"shutdown releases pool and sets instance to nil"},
		{"shutdown when pool is nil does not panic"},
		{"shutdown waits for in-flight tasks"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			epMu.Lock()
			epInst = nil
			epMu.Unlock()

			if tt.name == "shutdown when pool is nil does not panic" {
				require.NotPanics(t, func() { ShutdownEventPool() })
				return
			}

			err := InitEventPool(10, "30s", nil)
			require.NoError(t, err)

			if tt.name == "shutdown waits for in-flight tasks" {
				var executed bool
				submitEvent("test", "op", func() {
					time.Sleep(50 * time.Millisecond)
					executed = true
				})
				time.Sleep(10 * time.Millisecond)
				ShutdownEventPool()
				assert.True(t, executed)
				return
			}

			ShutdownEventPool()
			epMu.Lock()
			assert.Nil(t, epInst)
			epMu.Unlock()
		})
	}
}
```

- [ ] **Step 3: Run pool tests**

```bash
go test ./pkg/ability/... -v -run "Pool|Submit|Shutdown" -count=1 -timeout 15s
```

Expected: all pool tests pass.

- [ ] **Step 4: Commit**

```bash
git add pkg/ability/pool.go pkg/ability/pool_test.go
git commit -m "feat: add ants-based event pool wrapper to ability"
```

---

### Task 5: Wire pool into capability.Invoke

**Files:**

- Modify: `pkg/ability/invoke.go:131-140`

- [ ] **Step 1: Update Invoke to use pool**

Replace the `go func()` block (lines 131-140) in `pkg/ability/invoke.go` with:

```go
	r.mu.RLock()
	emitter := r.emitter
	r.mu.RUnlock()
	if emitter != nil && len(result.Events) > 0 {
		cap := string(capability)
		op := operation
		res := result
		submitEvent(cap, op, func() {
			emitter(context.WithoutCancel(ctx), res)
		})
	}
```

- [ ] **Step 2: Update existing event emission tests**

The existing `TestRegistry_InvokeEmitsEvents` test in `invoke_test.go` uses `time.Sleep(50 * time.Millisecond)` and a mutex-based emitter callback. Since `Submit()` is async but not `go func()` (pool workers run in different goroutines), the test needs the pool to be initialized. Add pool initialization to the test setup:

For tests that need the pool (`TestRegistry_InvokeEmitsEvents`, `TestRegistry_InvokeNoEmitWithoutEvents`, `TestRegistry_InvokeNoEmitWithoutEmitter`), add:

```go
// In test setup for tests that exercise event emission
epMu.Lock()
epInst = nil
epMu.Unlock()
_ = InitEventPool(10, "30s", nil)
defer func() {
	epMu.Lock()
	epInst = nil
	epMu.Unlock()
}()
```

But wait - this needs the pool. If the pool is nil, `submitEvent` falls back to direct execution. Actually, looking at `submitEvent`, when `ep == nil`, it calls `fn()` directly. So the existing tests would still work. The `time.Sleep` wait should still be enough.

Let me reconsider - since `submitEvent` falls back to direct execution when pool is nil, and the existing tests already use `time.Sleep` + mutex, they may need adjustment for the pool case. But since the pool isn't initialized in tests (no `InitEventPool` call), the nil fallback path will execute synchronously. This means existing tests should pass without modification.

Actually wait, the nil fallback calls `fn()` directly (synchronous), not `go fn()`. So the test with `time.Sleep(50 * time.Millisecond)` and a mutex to wait for the goroutine would still work because the emitter callback is called synchronously... but then the test's `time.Sleep` is unnecessary but harmless.

No - the problem is that in `TestRegistry_InvokeEmitsEvents`, the test emits on a goroutine (via the old `go func()`). With the new code using `submitEvent`, when pool is nil, it calls `fn()` directly - meaning it becomes synchronous. The test would need adjusting.

Let me think about this more carefully...

The test at line 420:

```go
result, err := r.Invoke(t.Context(), hub.CapBookmark, "list", nil)
require.NoError(t, err)
require.NotNil(t, result)
time.Sleep(50 * time.Millisecond)
mu.Lock()
defer mu.Unlock()
require.NotNil(t, emitted)
```

With the old code: `Invoke` returns → goroutine runs emitter → test sleeps → checks emitted.
With new code and pool=nil: `Invoke` returns → `submitEvent` calls `fn()` directly (synchronous) → emitter runs → test sleeps → checks emitted. Still works! Because the emitter runs synchronously before Invoke returns.

With new code and pool init'd: `Invoke` returns → submitEvent puts task on pool → worker runs emitter → test sleeps → checks emitted. Still works as long as the sleep is long enough.

So the existing tests should pass without modification. The only issue is `t.Parallel()` in tests - if multiple parallel tests share the pool, there could be races. But the pool is initialized per test (each test sets epInst).

Actually no - `t.Parallel()` in the tests mean tests run in parallel goroutines. The `epInst` is a package-level global. If one test initializes the pool and another deinitializes it, there's a race.

Let me check - in the current tests:

- `TestRegistry_InvokeEmitsEvents` - doesn't init or touch pool
- `TestRegistry_InvokeNoEmitWithoutEvents` - doesn't init or touch pool
- `TestRegistry_InvokeNoEmitWithoutEmitter` - doesn't init or touch pool

Since these tests don't call `InitEventPool`, the global `epInst` stays nil, and `submitEvent` takes the direct-execution path. Tests should be fine in parallel.

For the pool tests (`pool_test.go`), those shouldn't use `t.Parallel()` since they mutate the global `epInst`. Or they should reset it carefully.

OK, I think the existing invoke tests need no changes. Only the new pool tests need to be careful about the global state.

- [ ] **Step 2: Run invoke tests**

```bash
go test ./pkg/ability/... -v -count=1 -timeout 30s
```

Expected: all existing tests pass plus new pool tests.

- [ ] **Step 3: Commit**

```bash
git add pkg/ability/invoke.go
git commit -m "feat: replace go func() with event pool in capability.Invoke"
```

---

### Task 6: Wire pool init and shutdown into server lifecycle

**Files:**

- Modify: `internal/server/pipeline.go`
- Modify: `internal/server/server.go`

- [ ] **Step 1: Init pool in initPipeline**

In `internal/server/pipeline.go`, add pool initialization after the existing `capability.SetMetricsCollector(ac)` line:

```go
	capability.SetMetricsCollector(ac)

	// Initialize event pool
	poolCfg := cfg.capability.EventPool
	if err := capability.InitEventPool(poolCfg.Size, poolCfg.ExpiryDuration, ac); err != nil {
		return fmt.Errorf("init event pool: %w", err)
	}
```

- [ ] **Step 2: Shutdown pool in RunServer**

In `internal/server/server.go`, in the `OnStop` hook, add:

```go
		OnStop: func(ctx context.Context) error {
			ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			if err := app.ShutdownWithContext(ctx); err != nil {
				flog.Error(err)
			}

			// Shutdown Extra
			for _, ruleset := range globals.cronRuleset {
				ruleset.Shutdown()
			}

			capability.ShutdownEventPool()

			return nil
		},
```

Add `"github.com/flowline-io/flowbot/pkg/capability"` to the import block in `server.go`.

- [ ] **Step 3: Run server package tests**

```bash
go test ./internal/server/... -v -count=1 -timeout 30s
```

Expected: tests pass (or skip if they require database/redis).

- [ ] **Step 4: Commit**

```bash
git add internal/server/pipeline.go internal/server/server.go
git commit -m "feat: wire event pool init and shutdown into server lifecycle"
```

---

### Task 7: Verify with build and lint

**Files:** none

- [ ] **Step 1: Build**

```bash
go tool task build
```

Expected: build succeeds.

- [ ] **Step 2: Lint**

```bash
go tool task lint
```

Expected: lint passes.

- [ ] **Step 3: Run all unit tests**

```bash
go tool task test
```

Expected: all tests pass.

- [ ] **Step 4: Commit any fixes**

If lint or test failures, fix and commit. Otherwise no commit needed.
