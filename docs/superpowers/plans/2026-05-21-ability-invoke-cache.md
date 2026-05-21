# Ability Invoke Response Caching Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Cache Read operation results from `ability.Invoke()` in Ristretto with 2-minute TTL; write operations invalidate capability-level cache entries.

**Architecture:** Cache-aside wrapper inside `Registry.Invoke()`. Cache key = `ability:{capType}:{op}:{sha1(sortedParamsJSON)}`. Mutation detection via operation name substring matching against known write verbs. Active invalidation via `DelByPrefix` backed by a `sync.Map` key index. `sonic` for serialization. Cache failures never affect correctness.

**Tech Stack:** Go 1.26+, Ristretto v2, sonic, SHA1, sync.Map

---

### Task 1: Add IsMutation to operations.go

**Files:**
- Modify: `pkg/ability/operations.go` (append at end of file)

- [ ] **Step 1: Write the failing test**

Create the test in `pkg/ability/operations_test.go`. Read existing file first to understand structure.

```go
func TestIsMutation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		operation string
		want      bool
	}{
		{"list is read", "list", false},
		{"get is read", "get", false},
		{"search is read", "search", false},
		{"check_url is read", "check_url", false},
		{"list_tasks is read", "list_tasks", false},
		{"get_columns is read", "get_columns", false},
		{"create is mutation", "create", true},
		{"delete is mutation", "delete", true},
		{"update is mutation", "update", true},
		{"move_task is mutation", "move_task", true},
		{"archive is mutation", "archive", true},
		{"attach_tags is mutation", "attach_tags", true},
		{"detach_tags is mutation", "detach_tags", true},
		{"complete_task is mutation", "complete_task", true},
		{"mark_entry_read is mutation", "mark_entry_read", true},
		{"mark_entry_unread is mutation", "mark_entry_unread", true},
		{"star_entry is mutation", "star_entry", true},
		{"unstar_entry is mutation", "unstar_entry", true},
		{"send is mutation", "send", true},
		{"add is mutation", "add", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := IsMutation(tt.operation)
			assert.Equal(t, tt.want, got)
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./pkg/ability/ -run TestIsMutation -v
```
Expected: FAIL — `IsMutation` not defined.

- [ ] **Step 3: Write minimal implementation**

Append to `pkg/ability/operations.go`:

```go
import "strings"

var mutationVerbs = []string{
	"create", "delete", "update", "move",
	"archive", "attach", "detach", "complete",
	"mark_read", "mark_unread", "star", "unstar",
	"send", "add",
}

// IsMutation reports whether the operation name indicates a write that modifies state.
func IsMutation(op string) bool {
	for _, v := range mutationVerbs {
		if strings.Contains(op, v) {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./pkg/ability/ -run TestIsMutation -v
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/ability/operations.go pkg/ability/operations_test.go
git commit -m "feat: add IsMutation function for operation type classification"
```

---

### Task 2: Add GetBytes, SetBytesWithTTL, DelByPrefix with key index to cache.go

**Files:**
- Modify: `pkg/cache/cache.go` (add new fields and methods)
- Modify: `pkg/cache/cache_test.go` (add tests for new methods)

- [ ] **Step 1: Write the failing test**

Append to `pkg/cache/cache_test.go`:

```go
func TestCacheDelByPrefix(t *testing.T) {
	cache, err := NewCache(&config.Type{})
	require.NoError(t, err)
	require.NotNil(t, cache)
	defer cache.Wait()

	tests := []struct {
		name     string
		setup    func(c *Cache)
		prefix   string
		wantKeys map[string]bool // key -> should exist after DelByPrefix
	}{
		{
			name: "deletes all keys under prefix",
			setup: func(c *Cache) {
				c.SetWithTTLCap("ability:bookmark:list:abc", []byte("1"), 1, time.Hour, "bookmark")
				c.SetWithTTLCap("ability:bookmark:get:def", []byte("2"), 1, time.Hour, "bookmark")
			},
			prefix: "bookmark",
			wantKeys: map[string]bool{
				"ability:bookmark:list:abc": false,
				"ability:bookmark:get:def":  false,
			},
		},
		{
			name: "does not affect keys under different prefix",
			setup: func(c *Cache) {
				c.SetWithTTLCap("ability:bookmark:list:abc", []byte("1"), 1, time.Hour, "bookmark")
				c.SetWithTTLCap("ability:kanban:list:xyz", []byte("2"), 1, time.Hour, "kanban")
			},
			prefix: "bookmark",
			wantKeys: map[string]bool{
				"ability:bookmark:list:abc": false,
				"ability:kanban:list:xyz":   true,
			},
		},
		{
			name: "empty prefix is no-op",
			setup: func(c *Cache) {
				c.SetWithTTLCap("ability:bookmark:list:abc", []byte("1"), 1, time.Hour, "bookmark")
			},
			prefix: "",
			wantKeys: map[string]bool{
				"ability:bookmark:list:abc": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(cache)
			cache.Wait()

			cache.DelByPrefix(tt.prefix)
			cache.Wait()

			for key, wantExist := range tt.wantKeys {
				_, ok := cache.GetRaw(key)
				require.Equal(t, wantExist, ok, "key %s existence mismatch", key)
			}
		})
	}
}

func TestCacheGetBytes(t *testing.T) {
	cache, err := NewCache(&config.Type{})
	require.NoError(t, err)
	require.NotNil(t, cache)
	defer cache.Wait()

	tests := []struct {
		name      string
		key       string
		value     []byte
		wantValue []byte
		wantOK    bool
	}{
		{
			name:      "existing bytes value",
			key:       "bytes_key",
			value:     []byte("hello bytes"),
			wantValue: []byte("hello bytes"),
			wantOK:    true,
		},
		{
			name:      "empty bytes value",
			key:       "empty_bytes_key",
			value:     []byte{},
			wantValue: []byte{},
			wantOK:    true,
		},
		{
			name:      "missing key",
			key:       "nonexistent",
			value:     nil,
			wantValue: nil,
			wantOK:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != nil {
				cache.SetWithTTL(tt.key, tt.value, 1, time.Hour)
				cache.Wait()
			}

			got, ok := cache.GetBytes(tt.key)
			require.Equal(t, tt.wantOK, ok)
			require.Equal(t, tt.wantValue, got)
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./pkg/cache/ -run "TestCacheDelByPrefix|TestCacheGetBytes" -v
```
Expected: FAIL — methods not defined.

- [ ] **Step 3: Write minimal implementation**

Modify `pkg/cache/cache.go`. Add `sync` import if not already present (it already is). Replace the `Cache` struct and add new methods.

**Change the struct** (lines 15-17):

```go
type Cache struct {
	i        *ristretto.Cache[string, any]
	keyIndex sync.Map // capability_type -> *sync.Map (set of keys)
}
```

**Add new methods** after `Wait()` (line 52):

```go
// GetBytes retrieves raw bytes from the cache. Returns false if the key is not found
// or the value is not a byte slice.
func (c *Cache) GetBytes(key string) ([]byte, bool) {
	val, ok := c.i.Get(key)
	if !ok {
		return nil, false
	}
	b, ok := val.([]byte)
	if !ok {
		return nil, false
	}
	return b, true
}

// DelByPrefix removes all cached keys registered under the given capability prefix.
// The prefix corresponds to a capability type string (e.g. "bookmark", "kanban").
func (c *Cache) DelByPrefix(capType string) {
	if capType == "" {
		return
	}
	val, ok := c.keyIndex.LoadAndDelete(capType)
	if !ok {
		return
	}
	m := val.(*sync.Map)
	m.Range(func(key, _ any) bool {
		c.i.Del(key.(string))
		return true
	})
}

// registerKey adds a key to the capability prefix index for later prefix-based deletion.
func (c *Cache) registerKey(capType, key string) {
	actual, _ := c.keyIndex.LoadOrStore(capType, &sync.Map{})
	m := actual.(*sync.Map)
	m.Store(key, struct{}{})
}
```

**Modify `SetWithTTL`** (lines 38-40) to also support capability registration. Keep the existing signature unchanged but add an overload:

After `SetWithTTL` (line 40), add:

```go
// SetWithTTLCap stores a byte value with TTL and registers the key under the given
// capability prefix for later prefix-based invalidation via DelByPrefix.
func (c *Cache) SetWithTTLCap(key string, value []byte, cost int64, ttl time.Duration, capType string) bool {
	ok := c.i.SetWithTTL(key, value, cost, ttl)
	if ok {
		c.registerKey(capType, key)
	}
	return ok
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./pkg/cache/ -run "TestCacheDelByPrefix|TestCacheGetBytes" -v
```
Expected: PASS

- [ ] **Step 5: Run all cache tests**

```bash
go test ./pkg/cache/ -v
```
Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add pkg/cache/cache.go pkg/cache/cache_test.go
git commit -m "feat: add GetBytes, SetWithTTLCap, DelByPrefix with key index to cache"
```

---

### Task 3: Add cache-aside logic to Invoke()

**Files:**
- Modify: `pkg/ability/invoke.go` (insert cache logic in Invoke method)
- Modify: `pkg/ability/invoke_test.go` (add cache behavior tests)

- [ ] **Step 1: Write the failing test**

Append to `pkg/ability/invoke_test.go`:

```go
func TestRegistry_InvokeCacheHit(t *testing.T) {
	// Cannot use t.Parallel(): tests share cache.Instance
	_ = setupTestCache(t)

	tests := []struct {
		name string
	}{
		{"second call with same params returns cached result without invoking handler"},
		{"cache stores result data correctly on subsequent hit"},
		{"cache preserves text field on hit"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRegistry()
			callCount := 0
			err := r.Register(hub.CapBookmark, "list", func(_ context.Context, _ map[string]any) (*InvokeResult, error) {
				callCount++
				return &InvokeResult{Data: "result", Text: "cached text"}, nil
			})
			require.NoError(t, err)

			result1, err := r.Invoke(t.Context(), hub.CapBookmark, "list", map[string]any{"key": "val"})
			require.NoError(t, err)
			require.Equal(t, 1, callCount)

			result2, err := r.Invoke(t.Context(), hub.CapBookmark, "list", map[string]any{"key": "val"})
			require.NoError(t, err)
			require.Equal(t, 1, callCount, "handler should not be called again on cache hit")

			assert.Equal(t, "result", result1.Data)
			assert.Equal(t, "result", result2.Data)
			assert.Equal(t, "cached text", result2.Text)
		})
	}
}

func TestRegistry_InvokeCacheMiss(t *testing.T) {
	// Cannot use t.Parallel(): tests share cache.Instance
	_ = setupTestCache(t)

	tests := []struct {
		name string
	}{
		{"first call invokes handler and returns result"},
		{"different params produce different cache keys"},
		{"handler error is not cached"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRegistry()
			callCount := 0
			err := r.Register(hub.CapBookmark, "list", func(_ context.Context, _ map[string]any) (*InvokeResult, error) {
				callCount++
				if tt.name == "handler error is not cached" {
					return nil, errors.New("provider error")
				}
				return &InvokeResult{Data: "fresh"}, nil
			})
			require.NoError(t, err)

			result, err := r.Invoke(t.Context(), hub.CapBookmark, "list", map[string]any{"key": "first"})
			if tt.name == "handler error is not cached" {
				require.Error(t, err)
				require.Equal(t, 1, callCount)
				_, err = r.Invoke(t.Context(), hub.CapBookmark, "list", map[string]any{"key": "first"})
				require.Error(t, err)
				require.Equal(t, 2, callCount, "error should not be cached, handler called again")
				return
			}
			require.NoError(t, err)
			require.Equal(t, 1, callCount)
			assert.Equal(t, "fresh", result.Data)

			if tt.name == "different params produce different cache keys" {
				result2, err := r.Invoke(t.Context(), hub.CapBookmark, "list", map[string]any{"key": "second"})
				require.NoError(t, err)
				require.Equal(t, 2, callCount, "different params should be cache miss")
				assert.Equal(t, "fresh", result2.Data)
			}
		})
	}
}

func TestRegistry_InvokeCacheMutationInvalidates(t *testing.T) {
	// Cannot use t.Parallel(): tests share cache.Instance
	_ = setupTestCache(t)

	tests := []struct {
		name string
	}{
		{"write operation invalidates all cached read operations for same capability"},
		{"write on one capability does not affect another capability"},
		{"multiple writes invalidate progressively"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRegistry()
			listCallCount := 0
			err := r.Register(hub.CapBookmark, "list", func(_ context.Context, _ map[string]any) (*InvokeResult, error) {
				listCallCount++
				return &InvokeResult{Data: "list result"}, nil
			})
			require.NoError(t, err)

			if tt.name == "write on one capability does not affect another capability" {
				kanbanCallCount := 0
				err = r.Register(hub.CapKanban, "list_tasks", func(_ context.Context, _ map[string]any) (*InvokeResult, error) {
					kanbanCallCount++
					return &InvokeResult{Data: "kanban result"}, nil
				})
				require.NoError(t, err)
			}

			createCallCount := 0
			err = r.Register(hub.CapBookmark, "create", func(_ context.Context, _ map[string]any) (*InvokeResult, error) {
				createCallCount++
				return &InvokeResult{Text: "created"}, nil
			})
			require.NoError(t, err)

			_, err = r.Invoke(t.Context(), hub.CapBookmark, "list", nil)
			require.NoError(t, err)
			require.Equal(t, 1, listCallCount)

			_, err = r.Invoke(t.Context(), hub.CapBookmark, "create", nil)
			require.NoError(t, err)

			_, err = r.Invoke(t.Context(), hub.CapBookmark, "list", nil)
			require.NoError(t, err)
			require.Equal(t, 2, listCallCount, "cache should be invalidated after write")

			if tt.name == "write on one capability does not affect another capability" {
				_, err = r.Invoke(t.Context(), hub.CapKanban, "list_tasks", nil)
				require.NoError(t, err)
				_, err = r.Invoke(t.Context(), hub.CapKanban, "list_tasks", nil)
				require.NoError(t, err)
				require.Equal(t, 1, kanbanCallCount, "kanban cache should survive bookmark write")
			}

			if tt.name == "multiple writes invalidate progressively" {
				_, err = r.Invoke(t.Context(), hub.CapBookmark, "create", nil)
				require.NoError(t, err)
				_, err = r.Invoke(t.Context(), hub.CapBookmark, "list", nil)
				require.NoError(t, err)
				require.Equal(t, 3, listCallCount)
			}
		})
	}
}

func TestRegistry_InvokeCacheCursorSkip(t *testing.T) {
	// Cannot use t.Parallel(): tests share cache.Instance
	_ = setupTestCache(t)

	tests := []struct {
		name string
	}{
		{"cursor param bypasses cache read"},
		{"cursor param bypasses cache write"},
		{"non-cursor params are cached normally"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRegistry()
			callCount := 0
			err := r.Register(hub.CapBookmark, "list", func(_ context.Context, _ map[string]any) (*InvokeResult, error) {
				callCount++
				return &InvokeResult{Data: "result"}, nil
			})
			require.NoError(t, err)

			params := map[string]any{"cursor": "next_page_token"}
			if tt.name == "non-cursor params are cached normally" {
				params = map[string]any{"archived": true}
			}

			_, err = r.Invoke(t.Context(), hub.CapBookmark, "list", params)
			require.NoError(t, err)
			_, err = r.Invoke(t.Context(), hub.CapBookmark, "list", params)
			require.NoError(t, err)

			if tt.name == "non-cursor params are cached normally" {
				require.Equal(t, 1, callCount, "non-cursor params should be cached")
			} else {
				require.Equal(t, 2, callCount, "cursor params should not be cached")
			}
		})
	}
}

func TestRegistry_InvokeCacheSerializationRoundtrip(t *testing.T) {
	// Cannot use t.Parallel(): tests share cache.Instance
	_ = setupTestCache(t)

	t.Run("roundtrip preserves all result fields", func(t *testing.T) {
		r := NewRegistry()
		err := r.Register(hub.CapBookmark, "list", func(_ context.Context, _ map[string]any) (*InvokeResult, error) {
			return &InvokeResult{
				Data: map[string]any{"nested": "value"},
				Page: &PageInfo{HasMore: true},
				Text: "some text",
				Meta: map[string]any{"source": "cache"},
			}, nil
		})
		require.NoError(t, err)

		_, err = r.Invoke(t.Context(), hub.CapBookmark, "list", nil)
		require.NoError(t, err)

		result, err := r.Invoke(t.Context(), hub.CapBookmark, "list", nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Page.HasMore)
		assert.Equal(t, "some text", result.Text)
		assert.Equal(t, "cache", result.Meta["source"])
	})
}

func setupTestCache(t *testing.T) *cache.Cache {
	t.Helper()
	cache.Instance = nil
	c, err := cache.NewCache(&config.Type{})
	require.NoError(t, err)
	return c
}
```

The test file will need these additional imports at the top:

```go
import (
	// ... existing imports ...
	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/config"
)
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./pkg/ability/ -run "TestRegistry_InvokeCache" -v
```
Expected: FAIL — cache logic not yet implemented, or will PASS if cache.Instance is nil and logic gracefully skips.

First check: tests should compile but some assertions on cache behavior will fail because caching is not yet implemented.

- [ ] **Step 3: Write minimal implementation**

Modify `pkg/ability/invoke.go`. Add imports and helper functions, then modify `Invoke`.

**Add imports** (update the import block at lines 4-14):

```go
import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"sort"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"go.opentelemetry.io/otel/attribute"

	"github.com/flowline-io/flowbot/pkg/cache"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/metrics"
	"github.com/flowline-io/flowbot/pkg/trace"
	"github.com/flowline-io/flowbot/pkg/types"
)
```

**Add helper functions** before `RegisterInvoker` (after line 45):

```go
func buildCacheKey(capability hub.CapabilityType, operation string, params map[string]any) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	sorted := make(map[string]any, len(keys))
	for _, k := range keys {
		sorted[k] = params[k]
	}
	data, _ := sonic.MarshalString(sorted)
	h := sha1.New()
	_, _ = h.Write([]byte(data))
	hash := hex.EncodeToString(h.Sum(nil))
	return "ability:" + string(capability) + ":" + operation + ":" + hash
}

func hasCursorParam(params map[string]any) bool {
	_, ok := params["cursor"]
	return ok
}
```

**Modify `Invoke` method** (lines 73-139). Insert cache check after the invoker lookup (after line 87), and cache write + invalidation around the existing code:

Replace the entire `Invoke` method body (lines 73-139) with:

```go
func (r *Registry) Invoke(ctx context.Context, capability hub.CapabilityType, operation string, params map[string]any) (*InvokeResult, error) {
	if params == nil {
		params = map[string]any{}
	}
	r.mu.RLock()
	ops, ok := r.handlers[capability]
	if !ok {
		r.mu.RUnlock()
		return nil, types.Errorf(types.ErrNotFound, "capability %s not found", capability)
	}
	invoker, ok := ops[operation]
	r.mu.RUnlock()
	if !ok {
		return nil, types.Errorf(types.ErrNotImplemented, "operation %s.%s not implemented", capability, operation)
	}

	isMut := IsMutation(operation)
	skipCache := isMut || hasCursorParam(params)

	var cacheKey string
	if !skipCache && cache.Instance != nil {
		cacheKey = buildCacheKey(capability, operation, params)
		if cached, ok := cache.Instance.GetBytes(cacheKey); ok {
			var result InvokeResult
			if err := sonic.UnmarshalString(string(cached), &result); err == nil {
				return &result, nil
			}
		}
	}

	ctx, span := trace.StartSpan(ctx, "ability."+string(capability)+"."+operation,
		attribute.String("capability.name", string(capability)),
		attribute.String("capability.operation", operation),
	)
	defer span.End()

	start := time.Now()
	result, err := invoker(ctx, params)
	if err != nil {
		trace.RecordError(ctx, err)
		r.mu.RLock()
		mc := r.metrics
		r.mu.RUnlock()
		if mc != nil {
			mc.IncInvokeTotal(string(capability), operation, "error")
			mc.ObserveInvokeDuration(string(capability), operation, time.Since(start).Seconds())
			code := "unknown"
			if te, ok := err.(*types.Error); ok {
				code = te.Code
			}
			mc.IncInvokeError(string(capability), operation, code)
		}
		return nil, err
	}
	if result == nil {
		result = &InvokeResult{}
	}
	result.Capability = capability
	result.Operation = operation

	r.mu.RLock()
	mc := r.metrics
	r.mu.RUnlock()
	if mc != nil {
		mc.IncInvokeTotal(string(capability), operation, "ok")
		mc.ObserveInvokeDuration(string(capability), operation, time.Since(start).Seconds())
	}

	r.mu.RLock()
	emitter := r.emitter
	r.mu.RUnlock()
	if emitter != nil && len(result.Events) > 0 {
		capt := string(capability)
		op := operation
		res := result
		submitEvent(capt, op, func() {
			emitter(context.WithoutCancel(ctx), res)
		})
	}

	if !skipCache && cache.Instance != nil {
		if data, merr := sonic.MarshalString(result); merr == nil {
			cache.Instance.SetWithTTLCap(cacheKey, []byte(data), 1, cache.TTLShort.Duration(), string(capability))
		}
	}

	if isMut && cache.Instance != nil {
		cache.Instance.DelByPrefix(string(capability))
	}

	return result, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./pkg/ability/ -run "TestRegistry_InvokeCache" -v
```
Expected: all PASS

- [ ] **Step 5: Run all ability tests**

```bash
go test ./pkg/ability/ -v
```
Expected: all PASS (including existing tests)

- [ ] **Step 6: Run all cache tests**

```bash
go test ./pkg/cache/ -v
```
Expected: all PASS

- [ ] **Step 7: Run lint**

```bash
go tool task lint
```
Expected: clean (no new warnings)

- [ ] **Step 8: Commit**

```bash
git add pkg/ability/invoke.go pkg/ability/invoke_test.go
git commit -m "feat: add cache-aside layer to ability.Invoke with Ristretto"
```

---

### Task 4: Integration verification

- [ ] **Step 1: Build**

```bash
go tool task build
```
Expected: build succeeds

- [ ] **Step 2: Run full unit test suite**

```bash
go tool task test
```
Expected: all tests pass

- [ ] **Step 3: Verify no new lint violations**

```bash
go tool task lint
```
Expected: clean

- [ ] **Step 4: Commit (if any final adjustments)**

```bash
git add -A
git diff --cached --stat
git commit -m "chore: final integration verification for ability cache"
```
