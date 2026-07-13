# Health Dashboard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `/service/web/healthz` page with auto-refreshing system health metrics: DB/Redis latency, runtime stats, capability health, and recent error log summary.

**Architecture:** Extend `pkg/cache` and `internal/store` with Ping methods for direct health checks. Add a 50-entry ring buffer to `pkg/flog` for recent ERROR log retrieval. New templ page + partial with HTMX polling renders four DaisyUI card sections.

**Tech Stack:** Go 1.26+, templ v0.3, HTMX 2.x, DaisyUI v5, fiber v3, zerolog

---

### Task 1: Add Ping to cache.RedisStore and global accessor

**Files:**
- Modify: `pkg/cache/redis.go:190-212`
- New: `pkg/cache/cache.go` (if not exists, else modify)
- Modify: `internal/server/func.go:9-14`

- [ ] **Step 1: Add Ping method to RedisStore**

Add the Ping method at the end of `pkg/cache/redis.go` (before `toAny`, after line 190):

```go
// Ping checks Redis connectivity and returns the round-trip latency.
func (s *RedisStore) Ping(ctx context.Context) (time.Duration, error) {
	start := time.Now()
	err := s.client.Ping(ctx).Err()
	return time.Since(start), err
}
```

Also add `"time"` to the imports in `redis.go` (alongside existing `"context"` and `"fmt"`).

- [ ] **Step 2: Add global Redis store accessor in pkg/cache**

Add to `pkg/cache/cache.go` (after `var Instance *Cache` at line 14):

```go
var defaultRedisStore *RedisStore

// SetDefaultRedisStore sets the global Redis store for health checks and other
// cross-package access. Called once during server initialization.
func SetDefaultRedisStore(s *RedisStore) {
	defaultRedisStore = s
}

// DefaultRedisStore returns the global Redis store. May return nil before
// initialization.
func DefaultRedisStore() *RedisStore {
	return defaultRedisStore
}
```

- [ ] **Step 3: Update server init to set global Redis store**

In `internal/server/func.go`, update `SetCacheStore` to also set the global:

```go
func SetCacheStore(s *cache.RedisStore) {
	cacheStore = s
	cache.SetDefaultRedisStore(s)
}
```

- [ ] **Step 4: Run test to verify no breakage**

```bash
go build ./pkg/cache/... ./internal/server/...
```

- [ ] **Step 5: Commit**

```bash
git add pkg/cache/redis.go pkg/cache/cache.go internal/server/func.go
git commit -m "feat: add Ping to RedisStore and global accessor for health checks"
```

---

### Task 2: Add Ping to store.Adapter interface + PostgreSQL implementation

**Files:**
- Modify: `internal/store/store.go:195-210`
- Modify: `internal/store/postgres/adapter.go`

- [ ] **Step 1: Add Ping method to Adapter interface**

In `internal/store/store.go`, add to the `Adapter` interface after the `Stats()` line (after line 207):

```go
	// Ping checks database connectivity and returns the round-trip latency.
	Ping(ctx context.Context) (time.Duration, error)
```

- [ ] **Step 2: Implement Ping in postgres adapter**

In `internal/store/postgres/adapter.go`, find the `GetDB()` method and add `Ping` after it:

```go
// Ping checks PostgreSQL connectivity and returns the round-trip latency.
func (a *adapter) Ping(ctx context.Context) (time.Duration, error) {
	if a.db == nil {
		return 0, fmt.Errorf("postgres: database not initialized")
	}
	start := time.Now()
	err := a.db.PingContext(ctx)
	return time.Since(start), err
}
```

Check imports: ensure `"fmt"`, `"time"`, and `"context"` are imported in the adapter file.

- [ ] **Step 3: Check other adapter implementations**

Run `grep -r "type.*adapter struct" internal/store/` to find all adapter implementations. If there are multiple (e.g., SQLite mock for tests), add a stub `Ping` method to each that returns `0, fmt.Errorf("not supported")`.

Search command:
```bash
rg "type \w+ struct" --glob 'internal/store/*/adapter*.go' -H
rg ":=\s*adapter\{" --glob 'internal/store/*/*.go' -H
```

- [ ] **Step 4: Run build to verify compilation**

```bash
go build ./internal/store/...
```

- [ ] **Step 5: Commit**

```bash
git add internal/store/store.go internal/store/postgres/adapter.go
git commit -m "feat: add Ping to store.Adapter and postgres implementation"
```

---

### Task 3: Add ring buffer to flog for recent ERROR logs

**Files:**
- Modify: `pkg/flog/flog.go`
- Create: `pkg/flog/flog_test.go` (add test cases)

- [ ] **Step 1: Write the failing test for ErrorEntry type and RecentErrors**

Add to `pkg/flog/flog_test.go`:

```go
import (
	"fmt"
	"sync"

	"github.com/stretchr/testify/require"
)

func TestRecentErrors(t *testing.T) {
	t.Run("empty buffer returns empty slice", func(t *testing.T) {
		ClearErrorBuffer()
		entries := RecentErrors()
		require.Empty(t, entries)
	})

	t.Run("captures error entries", func(t *testing.T) {
		ClearErrorBuffer()
		recordError(fmt.Errorf("test error 1"))
		recordError(fmt.Errorf("test error 2"))
		entries := RecentErrors()
		require.Len(t, entries, 2)
		require.Equal(t, "test error 1", entries[0].Message)
		require.Equal(t, "test error 2", entries[1].Message)
		require.NotZero(t, entries[0].Time)
	})

	t.Run("ring buffer wraps at capacity", func(t *testing.T) {
		ClearErrorBuffer()
		for i := 0; i < 55; i++ {
			recordError(fmt.Errorf("error %d", i))
		}
		entries := RecentErrors()
		require.Len(t, entries, errorBufferCapacity)
		// Oldest entries dropped, newest preserved
		require.Contains(t, entries[0].Message, "error 5")
		require.Contains(t, entries[errorBufferCapacity-1].Message, "error 54")
	})

	t.Run("thread-safe concurrent writes", func(t *testing.T) {
		ClearErrorBuffer()
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 10; j++ {
					recordError(fmt.Errorf("goroutine %d error %d", id, j))
				}
			}(i)
		}
		wg.Wait()
		entries := RecentErrors()
		require.NotEmpty(t, entries)
	})

	t.Run("caller field populated", func(t *testing.T) {
		ClearErrorBuffer()
		recordError(fmt.Errorf("caller test"))
		entries := RecentErrors()
		require.Len(t, entries, 1)
		require.Contains(t, entries[0].Caller, "flog_test.go")
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./pkg/flog/ -run TestRecentErrors -v
```
Expected: FAIL - `recordError`, `ClearErrorBuffer`, `RecentErrors`, `errorBufferCapacity` not defined.

- [ ] **Step 3: Implement ring buffer in flog.go**

Add after the existing `var` block (around line 32) in `pkg/flog/flog.go`:

```go
// ErrorEntry represents a recorded error log entry for health dashboard display.
type ErrorEntry struct {
	Time    time.Time `json:"time"`
	Message string    `json:"message"`
	Caller  string    `json:"caller,omitzero"`
}

const errorBufferCapacity = 50

var (
	errorBuf    = make([]ErrorEntry, 0, errorBufferCapacity)
	errorBufMu  sync.Mutex
	errorBufPos int
)

// recordError adds an error entry to the in-memory ring buffer.
func recordError(err error) {
	entry := ErrorEntry{
		Time:    time.Now(),
		Message: err.Error(),
	}
	// Capture caller: skip recordError, Err/Error, and the original caller.
	_, file, line, ok := runtime.Caller(3)
	if ok {
		entry.Caller = filepath.Base(file) + ":" + strconv.Itoa(line)
	}

	errorBufMu.Lock()
	defer errorBufMu.Unlock()

	if len(errorBuf) < errorBufferCapacity {
		errorBuf = append(errorBuf, entry)
	} else {
		errorBuf[errorBufPos%errorBufferCapacity] = entry
	}
	errorBufPos++
}

// RecentErrors returns a copy of recent error entries in insertion order.
func RecentErrors() []ErrorEntry {
	errorBufMu.Lock()
	defer errorBufMu.Unlock()

	if len(errorBuf) < errorBufferCapacity {
		result := make([]ErrorEntry, len(errorBuf))
		copy(result, errorBuf)
		return result
	}
	// Ring buffer full: return in order from oldest to newest.
	start := errorBufPos % errorBufferCapacity
	result := make([]ErrorEntry, 0, errorBufferCapacity)
	for i := 0; i < errorBufferCapacity; i++ {
		idx := (start + i) % errorBufferCapacity
		result = append(result, errorBuf[idx])
	}
	return result
}

// ClearErrorBuffer clears the error buffer. Exported for tests.
func ClearErrorBuffer() {
	errorBufMu.Lock()
	defer errorBufMu.Unlock()
	errorBuf = make([]ErrorEntry, 0, errorBufferCapacity)
	errorBufPos = 0
}
```

Update imports in `flog.go` to add `"runtime"`, `"strconv"`, and `"path/filepath"`.

- [ ] **Step 4: Update Err() function to call recordError**

In `flog.go`, modify the `Err` function (line 412-423) adding `recordError(err)` at the start:

```go
func Err(err error) {
	recordError(err)
	stateMu.RLock()
	evt := l.Error().Err(err)
	stateMu.RUnlock()
	if mustCaller() {
		evt = evt.Caller(1)
	}
	if mustStack() {
		evt = evt.Stack()
	}
	evt.Msg("error occurred")
}
```

The `recordError` is called BEFORE the mutex lock so that ring buffer operations don't block log writes.

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test ./pkg/flog/ -run TestRecentErrors -v
```
Expected: PASS

- [ ] **Step 6: Run full flog test suite**

```bash
go test ./pkg/flog/ -v
```
Expected: all existing + new tests pass.

- [ ] **Step 7: Commit**

```bash
git add pkg/flog/flog.go pkg/flog/flog_test.go
git commit -m "feat: add ring buffer for recent ERROR logs in flog"
```

---

### Task 4: Create healthz page template

**Files:**
- Create: `pkg/views/pages/healthz.templ`

- [ ] **Step 1: Write the page template**

Create `pkg/views/pages/healthz.templ`:

```templ
package pages

import (
	"github.com/flowline-io/flowbot/pkg/views/layout"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

templ HealthzPage(data HealthzData) {
	@layout.Base("Health — Flowbot") {
		<div class="flex items-center justify-between mb-6">
			<h1 class="text-2xl font-bold text-base-content">System Health</h1>
			<span class="text-sm text-base-content/50">
				Auto-refreshes every 30s
			</span>
		</div>
		@partials.HealthzStatus(data)
	}
}
```

Note: `HealthzData` type and `partials.HealthzStatus` will be defined in the next tasks.

- [ ] **Step 2: Commit**

```bash
git add pkg/views/pages/healthz.templ
git commit -m "feat: add healthz page template"
```

---

### Task 5: Create healthz status partial template

**Files:**
- Create: `pkg/views/partials/healthz_status.templ`

- [ ] **Step 1: Write the partial template**

Create `pkg/views/partials/healthz_status.templ`:

```templ
package partials

import (
	"time"
	"github.com/flowline-io/flowbot/pkg/flog"
)

// HealthzData is the data model for the health dashboard.
type HealthzData struct {
	PostgresLatency   time.Duration
	PostgresOk        bool
	RedisLatency      time.Duration
	RedisOk           bool
	Goroutines        int
	HeapAlloc         uint64
	TotalAlloc        uint64
	SysMem            uint64
	NumGC             uint32
	LastGCPause       time.Duration
	Capabilities      []HealthzCap
	Errors             []flog.ErrorEntry
}

// HealthzCap represents a capability health status for display.
type HealthzCap struct {
	Name   string
	Type   string
	Status string // "healthy", "unhealthy", "timeout", "na"
	Error  string
}

templ HealthzStatus(data HealthzData) {
	<div id="healthz-status"
		data-testid="healthz-status"
		hx-get="/service/web/healthz"
		hx-trigger="every 30s"
		hx-swap="outerHTML">
		<div class="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
			<!-- DB Latency -->
			<div class="card bg-base-100 shadow-sm" data-testid="healthz-db-latency">
				<div class="card-body">
					<h2 class="card-title text-base">Database Latency</h2>
					<div class="grid grid-cols-2 gap-4">
						<div class="stat">
							<div class="stat-title text-xs">PostgreSQL</div>
							<div class="stat-value text-lg">
								if data.PostgresOk {
									{ formatDuration(data.PostgresLatency) }
								} else {
									<span class="text-error">unreachable</span>
								}
							</div>
							<div class="stat-desc text-xs">
								if data.PostgresOk {
									<span class="text-success">connected</span>
								} else {
									<span class="text-error">disconnected</span>
								}
							</div>
						</div>
						<div class="stat">
							<div class="stat-title text-xs">Redis</div>
							<div class="stat-value text-lg">
								if data.RedisOk {
									{ formatDuration(data.RedisLatency) }
								} else {
									<span class="text-error">unreachable</span>
								}
							</div>
							<div class="stat-desc text-xs">
								if data.RedisOk {
									<span class="text-success">connected</span>
								} else {
									<span class="text-error">disconnected</span>
								}
							</div>
						</div>
					</div>
				</div>
			</div>

			<!-- Runtime -->
			<div class="card bg-base-100 shadow-sm" data-testid="healthz-runtime">
				<div class="card-body">
					<h2 class="card-title text-base">Runtime</h2>
					<div class="grid grid-cols-2 gap-4">
						<div class="stat">
							<div class="stat-title text-xs">Goroutines</div>
							<div class="stat-value text-lg">{ data.Goroutines }</div>
						</div>
						<div class="stat">
							<div class="stat-title text-xs">Heap Alloc</div>
							<div class="stat-value text-lg">{ formatBytes(data.HeapAlloc) }</div>
						</div>
						<div class="stat">
							<div class="stat-title text-xs">Total Alloc</div>
							<div class="stat-value text-lg">{ formatBytes(data.TotalAlloc) }</div>
						</div>
						<div class="stat">
							<div class="stat-title text-xs">Sys Memory</div>
							<div class="stat-value text-lg">{ formatBytes(data.SysMem) }</div>
						</div>
						<div class="stat">
							<div class="stat-title text-xs">GC Cycles</div>
							<div class="stat-value text-lg">{ data.NumGC }</div>
						</div>
						<div class="stat">
							<div class="stat-title text-xs">Last GC Pause</div>
							<div class="stat-value text-lg">{ formatDuration(data.LastGCPause) }</div>
						</div>
					</div>
				</div>
			</div>
		</div>

		<div class="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
			<!-- Capability Status -->
			<div class="card bg-base-100 shadow-sm" data-testid="healthz-capabilities">
				<div class="card-body">
					<h2 class="card-title text-base">Capability Status</h2>
					<div class="overflow-x-auto max-h-64 overflow-y-auto">
						<table class="table table-sm">
							<thead>
								<tr>
									<th>Capability</th>
									<th>Backend</th>
									<th>Status</th>
								</tr>
							</thead>
							<tbody>
								for _, cap := range data.Capabilities {
									<tr>
										<td class="font-medium">{ cap.Name }</td>
										<td class="text-base-content/70 text-xs">{ cap.Type }</td>
										<td>
											if cap.Status == "healthy" {
												<span class="badge badge-success badge-sm">healthy</span>
											} else if cap.Status == "timeout" {
												<span class="badge badge-warning badge-sm">timeout</span>
											} else if cap.Status == "na" {
												<span class="badge badge-ghost badge-sm">n/a</span>
											} else {
												<span class="badge badge-error badge-sm">{ cap.Status }</span>
											}
										</td>
									</tr>
								}
								if len(data.Capabilities) == 0 {
									<tr>
										<td colspan="3" class="text-center text-base-content/50">No capabilities registered</td>
									</tr>
								}
							</tbody>
						</table>
					</div>
				</div>
			</div>

			<!-- Recent Errors -->
			<div class="card bg-base-100 shadow-sm" data-testid="healthz-errors">
				<div class="card-body">
					<h2 class="card-title text-base">Recent Errors</h2>
					<div class="overflow-x-auto max-h-64 overflow-y-auto">
						if len(data.Errors) == 0 {
							<p class="text-base-content/50 text-sm">No recent errors</p>
						} else {
							<ul class="list-disc list-inside space-y-1">
								for _, e := range data.Errors {
									<li class="text-xs text-error">
										<span class="text-base-content/50">{ e.Time.Format("15:04:05") }</span>
										{" "}
										{ e.Message }
									</li>
								}
							</ul>
						}
					</div>
				</div>
			</div>
		</div>
	</div>
}
```

Note: templ functions `formatDuration` and `formatBytes` need helper functions. Add them in the next step.

- [ ] **Step 2: Add Go helper functions**

Append to `pkg/views/partials/helpers.go` (file exists with pagination and config helpers already):

```go
package partials

import (
	"fmt"
	"time"
)

// formatDuration formats a duration for display in the health dashboard.
func formatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%.2fµs", float64(d.Microseconds())+float64(d.Nanoseconds()%1000)/1000)
	}
	if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Milliseconds())+float64(d.Microseconds()%1000)/1000)
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

// formatBytes formats byte count for display.
func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
```

- [ ] **Step 3: Generate templ Go code**

```bash
templ generate pkg/views/pages/healthz.templ pkg/views/partials/healthz_status.templ
```

- [ ] **Step 4: Commit**

```bash
git add pkg/views/partials/healthz_status.templ pkg/views/partials/helpers.go pkg/views/pages/healthz.templ pkg/views/pages/healthz_templ.go pkg/views/partials/healthz_status_templ.go
git commit -m "feat: add healthz status partial and page templates"
```

---

### Task 6: Add healthzPage handler in web module

**Files:**
- Modify: `internal/modules/web/healthz_webservice.go`

- [ ] **Step 1: Write the handler function**

Add to `internal/modules/web/healthz_webservice.go` (at end of file):

```go
import (
	"runtime"
	"sync"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/hub"
)

// HealthzCap is the capability health info for the healthz template.
// (useful as a local type to avoid coupling the partial to hub types)
type healthzCapInfo struct {
	Name   string
	Type   string
	Status string
	Error  string
}

// healthzPage renders the system health dashboard.
func healthzPage(ctx fiber.Ctx) error {
	hctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	data := gatherHealthzData(hctx)

	ctx.Type("html")
	if ctx.Get("HX-Request") != "" {
		return partials.HealthzStatus(data).Render(context.Background(), ctx.Response().BodyWriter())
	}
	return pages.HealthzPage(data).Render(context.Background(), ctx.Response().BodyWriter())
}

func gatherHealthzData(ctx context.Context) partials.HealthzData {
	data := partials.HealthzData{}

	// PostgreSQL ping
	if store.Database != nil && store.Database.IsOpen() {
		latency, err := store.Database.Ping(ctx)
		data.PostgresLatency = latency
		data.PostgresOk = err == nil
	}

	// Redis ping
	if rs := cache.DefaultRedisStore(); rs != nil {
		latency, err := rs.Ping(ctx)
		data.RedisLatency = latency
		data.RedisOk = err == nil
	}

	// Runtime metrics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	data.Goroutines = runtime.NumGoroutine()
	data.HeapAlloc = memStats.HeapAlloc
	data.TotalAlloc = memStats.TotalAlloc
	data.SysMem = memStats.Sys
	data.NumGC = memStats.NumGC
	if memStats.NumGC > 0 {
		data.LastGCPause = time.Duration(memStats.PauseNs[(memStats.NumGC+255)%256])
	}

	// Capability health
	descriptors := hub.Default.List()
	results := make(chan healthzCapInfo, len(descriptors))
	var wg sync.WaitGroup

	for _, desc := range descriptors {
		wg.Add(1)
		go func(d hub.Descriptor) {
			defer wg.Done()
			capCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()

			info := healthzCapInfo{
				Name: string(d.Type),
				Type: string(d.Backend),
			}

			result, err := capability.Invoke(capCtx, d.Type, "health", map[string]any{})
			if err != nil {
				info.Status = "unhealthy"
				info.Error = err.Error()
			} else if result != nil && result.Data != nil {
				if ok, isBool := result.Data.(bool); isBool && ok {
					info.Status = "healthy"
				} else {
					info.Status = "unhealthy"
				}
			} else {
				info.Status = "na"
			}
			results <- info
		}(desc)
	}
	wg.Wait()
	close(results)
	for info := range results {
		data.Capabilities = append(data.Capabilities, partials.HealthzCap{
			Name:   info.Name,
			Type:   info.Type,
			Status: info.Status,
			Error:  info.Error,
		})
	}

	// Recent errors (last 10)
	allErrors := flog.RecentErrors()
	start := 0
	if len(allErrors) > 10 {
		start = len(allErrors) - 10
	}
	data.Errors = allErrors[start:]

	return data
}
```

Check existing imports in `webservice.go` to avoid duplicates. The file already imports `context`, `time`, `sync`(? - check), `runtime`, `store`, `pages`, `partials`, `flog`, etc. Add missing imports.

- [ ] **Step 2: Add route rule**

In `internal/modules/web/healthz_webservice.go`, define `healthzWebserviceRules`; register it in `rules.go` (`allWebserviceRules`):

```go
	webservice.Get("/healthz", healthzPage, route.WithNotAuth()),
```

- [ ] **Step 3: Run build to verify compilation**

```bash
go build ./internal/modules/web/...
```
Expected: pass

- [ ] **Step 4: Commit**

```bash
git add internal/modules/web/healthz_webservice.go internal/modules/web/rules.go
git commit -m "feat: add healthzPage handler and route for health dashboard"
```

---

### Task 7: Add nav link to base layout

**Files:**
- Modify: `pkg/views/layout/base.templ:38-39`

- [ ] **Step 1: Add Health nav link**

In `pkg/views/layout/base.templ`, after the "Capabilities" link (line 38), add:

```html
				<a href="/service/web/healthz" data-testid="nav-healthz" class="btn btn-ghost btn-sm">Health</a>
```

Insert it between line 38 (`<a ... Capabilities</a>`) and line 39 (theme toggle dropdown div).

- [ ] **Step 2: Regenerate templ code**

```bash
templ generate pkg/views/layout/base.templ
```

- [ ] **Step 3: Verify compilation**

```bash
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add pkg/views/layout/base.templ pkg/views/layout/base_templ.go
git commit -m "feat: add Health nav link to base layout"
```

---

### Task 8: Run format, lint, and full test suite

- [ ] **Step 1: Format code**

```bash
go tool task format
```

- [ ] **Step 2: Lint**

```bash
go tool task lint
```

- [ ] **Step 3: Unit tests**

```bash
go tool task test
```

- [ ] **Step 4: Fix any issues found**

Fix and re-run until clean.

- [ ] **Step 5: Commit if changes**

```bash
git add -A
git commit -m "chore: format, lint, and test fixes for health dashboard"
```

---

### Task 9: Write BDD spec for health dashboard

**Files:**
- Create: `tests/specs/healthz_spec_test.go`

- [ ] **Step 1: Write the BDD spec**

Create `tests/specs/healthz_spec_test.go`:

```go
package specs_test

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Health Dashboard /healthz", func() {
	var client *http.Client
	var baseURL string

	BeforeEach(func() {
		client = newHTTPClient()
		baseURL = appBaseURL()
	})

	It("returns 401 without authentication", func() {
		resp, err := client.Get(baseURL + "/service/web/healthz")
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
	})

	It("returns 200 with valid authentication", func() {
		req, _ := http.NewRequest("GET", baseURL+"/service/web/healthz", nil)
		setAuthCookie(req)
		resp, err := client.Do(req)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
	})

	It("renders all four metric sections", func() {
		req, _ := http.NewRequest("GET", baseURL+"/service/web/healthz", nil)
		setAuthCookie(req)
		resp, err := client.Do(req)
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		body := readBody(resp)
		Expect(body).To(ContainSubstring("Database Latency"))
		Expect(body).To(ContainSubstring("PostgreSQL"))
		Expect(body).To(ContainSubstring("Redis"))
		Expect(body).To(ContainSubstring("Runtime"))
		Expect(body).To(ContainSubstring("Goroutines"))
		Expect(body).To(ContainSubstring("Capability Status"))
		Expect(body).To(ContainSubstring("Recent Errors"))
	})

	It("has HTMX auto-refresh on the status section", func() {
		req, _ := http.NewRequest("GET", baseURL+"/service/web/healthz", nil)
		setAuthCookie(req)
		resp, err := client.Do(req)
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		body := readBody(resp)
		Expect(body).To(ContainSubstring("hx-trigger=\"every 30s\""))
		Expect(body).To(ContainSubstring("hx-get=\"/service/web/healthz\""))
	})
})
```

Note: Import the test helper functions that match the BDD suite conventions (like `newHTTPClient`, `appBaseURL`, `setAuthCookie`, `readBody`). Check existing spec files in `tests/specs/` for the exact helper names.

- [ ] **Step 2: Run BDD tests**

```bash
go tool task test:specs
```

- [ ] **Step 3: Fix any failures and re-run**

- [ ] **Step 4: Commit**

```bash
git add tests/specs/healthz_spec_test.go
git commit -m "test: add BDD spec for health dashboard /healthz"
```

---

### Task 10: Final verification

- [ ] **Step 1: Full build**

```bash
go tool task build
```

- [ ] **Step 2: Full test suite**

```bash
go tool task test
go tool task test:specs:ci
```

- [ ] **Step 3: Final lint**

```bash
go tool task lint
```

- [ ] **Step 4: Commit any remaining changes**

```bash
git status
git add -A
git commit -m "chore: final verification fixes for health dashboard"
```
