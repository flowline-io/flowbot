# Redis Cache Strategy: Unified Abstraction Layer — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a unified cache abstraction layer (`pkg/cache/`) with standardized key naming, TTL policies, metrics instrumentation, and migrate all existing Redis/Ristretto callers to the new interfaces.

**Architecture:** `StringCache`, `IntCache`, `SetCache`, `ListCache` interfaces implemented by `RistrettoStore` (in-process) and `RedisStore` (Redis). All operations auto-record Prometheus hit/miss/eviction/size metrics. Key type enforces `{prefix}:{entity}:{identifier}` naming. Three-phase migration: infrastructure → modules → cleanup.

**Tech Stack:** Go 1.26+, Ristretto v2, go-redis/v9, Prometheus client_golang, testify/require, uber/fx

**Spec:** `docs/superpowers/specs/2026-05-19-redis-cache-strategy-design.md`

---

## File Structure

```
pkg/cache/
  ├── cache.go           # RistrettoStore — enhanced existing Cache struct (MODIFY)
  ├── redis.go           # RedisStore — wraps rdb.Client, implements all interfaces (CREATE)
  ├── key.go             # Key builder type (CREATE)
  ├── ttl.go             # TTL policy constants (CREATE)
  ├── metrics.go         # Metric recording helpers (CREATE)
  ├── types.go           # StringCache, IntCache, SetCache, ListCache interfaces (CREATE)
  ├── cache_test.go      # Existing Ristretto tests (MODIFY)
  ├── redis_test.go      # RedisStore tests (CREATE)
  ├── key_test.go        # Key builder tests (CREATE)
  └── ttl_test.go        # TTL tests (CREATE)

pkg/stats/
  └── stats.go           # Add cache metric constants + getters (MODIFY)

internal/server/
  └── fx.go              # Register NewRedisStore in DI (MODIFY)

internal/server/
  └── func.go            # Migrate chat + online (MODIFY)

internal/modules/server/
  └── cron.go            # Migrate online count (MODIFY)

pkg/crawler/
  └── crawler.go         # Migrate sent/todo/sendtime (MODIFY)

pkg/types/ruleset/cron/
  └── cron.go            # Migrate filter set (MODIFY)

pkg/rdb/
  └── metrics.go         # Deprecate, logic moved to RedisStore (MODIFY)
  └── unique.go          # Deprecate, logic moved to RedisStore (MODIFY)

pkg/notify/rules/
  └── engine.go          # Accept RedisStore instead of *redis.Client (MODIFY)
  └── throttle.go        # Use IntCache (MODIFY)
  └── aggregate.go       # Use ListCache + StringCache.SetNX (MODIFY)

pkg/alarm/
  └── alarm.go           # Use RistrettoStore.StringCache (MODIFY)
```

---

## Phase 1: Infrastructure

### Task 1: TTL constants (`pkg/cache/ttl.go`)

**Files:**
- Create: `pkg/cache/ttl.go`
- Create: `pkg/cache/ttl_test.go`

- [ ] **Step 1: Write the test**

```go
package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTTLDuration(t *testing.T) {
	tests := []struct {
		name     string
		ttl      TTL
		wantDur  time.Duration
	}{
		{
			name:    "TTLNone is zero",
			ttl:     TTLNone,
			wantDur: 0,
		},
		{
			name:    "TTLMinute is one minute",
			ttl:     TTLMinute,
			wantDur: time.Minute,
		},
		{
			name:    "TTLShort is two minutes",
			ttl:     TTLShort,
			wantDur: 2 * time.Minute,
		},
		{
			name:    "TTLMedium is ten minutes",
			ttl:     TTLMedium,
			wantDur: 10 * time.Minute,
		},
		{
			name:    "TTLLong is one hour",
			ttl:     TTLLong,
			wantDur: time.Hour,
		},
		{
			name:    "TTLSession is 24 hours",
			ttl:     TTLSession,
			wantDur: 24 * time.Hour,
		},
		{
			name:    "TTLDay is 24 hours",
			ttl:     TTLDay,
			wantDur: 24 * time.Hour,
		},
		{
			name:    "TTLWeek is 7 days",
			ttl:     TTLWeek,
			wantDur: 7 * 24 * time.Hour,
		},
		{
			name:    "TTLMonth is 30 days",
			ttl:     TTLMonth,
			wantDur: 30 * 24 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.wantDur, tt.ttl.Duration())
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/cache/ -run TestTTLDuration -v`
Expected: FAIL (type `TTL` not defined)

- [ ] **Step 3: Write implementation**

```go
package cache

import "time"

type TTL time.Duration

const (
	TTLNone    TTL = 0
	TTLMinute  TTL = TTL(time.Minute)
	TTLShort   TTL = TTL(2 * time.Minute)
	TTLMedium  TTL = TTL(10 * time.Minute)
	TTLLong    TTL = TTL(1 * time.Hour)
	TTLSession TTL = TTL(24 * time.Hour)
	TTLDay     TTL = TTL(24 * time.Hour)
	TTLWeek    TTL = TTL(7 * 24 * time.Hour)
	TTLMonth   TTL = TTL(30 * 24 * time.Hour)
)

func (t TTL) Duration() time.Duration {
	return time.Duration(t)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/cache/ -run TestTTLDuration -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/cache/ttl.go pkg/cache/ttl_test.go
git commit -m "feat(cache): add TTL policy constants"
```

---

### Task 2: Key builder (`pkg/cache/key.go`)

**Files:**
- Create: `pkg/cache/key.go`
- Create: `pkg/cache/key_test.go`

- [ ] **Step 1: Write the test**

```go
package cache

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewKey(t *testing.T) {
	tests := []struct {
		name       string
		prefix     string
		entity     string
		identifier string
		want       string
	}{
		{
			name:       "online agent key",
			prefix:     "online",
			entity:     "agent",
			identifier: "host123",
			want:       "online:agent:host123",
		},
		{
			name:       "chat session key",
			prefix:     "chat",
			entity:     "session",
			identifier: "user456",
			want:       "chat:session:user456",
		},
		{
			name:       "notify throttle with compound identifier",
			prefix:     "notify",
			entity:     "throttle",
			identifier: "rule1:eventA:slack",
			want:       "notify:throttle:rule1:eventA:slack",
		},
		{
			name:       "cron filter key",
			prefix:     "cron",
			entity:     "filter",
			identifier: "job1:uid123",
			want:       "cron:filter:job1:uid123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := NewKey(tt.prefix, tt.entity, tt.identifier)
			require.Equal(t, tt.want, k.String())
		})
	}
}

func TestKeyString(t *testing.T) {
	tests := []struct {
		name string
		key  Key
		want string
	}{
		{
			name: "Key with all fields",
			key:  Key{Prefix: "a", Entity: "b", Identifier: "c"},
			want: "a:b:c",
		},
		{
			name: "Key with empty identifier",
			key:  Key{Prefix: "a", Entity: "b", Identifier: ""},
			want: "a:b:",
		},
		{
			name: "Key with empty entity",
			key:  Key{Prefix: "a", Entity: "", Identifier: "c"},
			want: "a::c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.key.String())
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/cache/ -run "TestNewKey|TestKeyString" -v`
Expected: FAIL

- [ ] **Step 3: Write implementation**

```go
package cache

import "fmt"

type Key struct {
	Prefix     string
	Entity     string
	Identifier string
}

func NewKey(prefix, entity, identifier string) Key {
	return Key{Prefix: prefix, Entity: entity, Identifier: identifier}
}

func (k Key) String() string {
	return fmt.Sprintf("%s:%s:%s", k.Prefix, k.Entity, k.Identifier)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/cache/ -run "TestNewKey|TestKeyString" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/cache/key.go pkg/cache/key_test.go
git commit -m "feat(cache): add Key builder with standardized naming"
```

---

### Task 3: Cache metrics in `pkg/stats`

**Files:**
- Modify: `pkg/stats/stats.go`

- [ ] **Step 1: Add metric constants and getters**

Append to `pkg/stats/stats.go` after the last `func DockerContainerTotalCounter()`:

```go
const (
	CacheHitTotalName      = "cache_hit_total"
	CacheMissTotalName     = "cache_miss_total"
	CacheEvictionTotalName = "cache_eviction_total"
	CacheSizeBytesName     = "cache_size_bytes"
)

func CacheHitTotalCounter(backend string) MetricInterface {
	return getOrCreateMetric(CacheHitTotalName, prometheus.Labels{"backend": backend})
}

func CacheMissTotalCounter(backend string) MetricInterface {
	return getOrCreateMetric(CacheMissTotalName, prometheus.Labels{"backend": backend})
}

func CacheEvictionTotalCounter(backend string) MetricInterface {
	return getOrCreateMetric(CacheEvictionTotalName, prometheus.Labels{"backend": backend})
}

func CacheSizeBytesGauge(backend string) MetricInterface {
	return getOrCreateMetric(CacheSizeBytesName, prometheus.Labels{"backend": backend})
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./pkg/stats/`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add pkg/stats/stats.go
git commit -m "feat(stats): add cache hit/miss/eviction/size metrics"
```

---

### Task 4: Metric recording helpers (`pkg/cache/metrics.go`)

**Files:**
- Create: `pkg/cache/metrics.go`

- [ ] **Step 1: Write the helper**

```go
package cache

import "github.com/flowline-io/flowbot/pkg/stats"

func recordHit(backend string) {
	stats.CacheHitTotalCounter(backend).Inc()
}

func recordMiss(backend string) {
	stats.CacheMissTotalCounter(backend).Inc()
}

func recordEviction(backend string) {
	stats.CacheEvictionTotalCounter(backend).Inc()
}

func recordSizeBytes(backend string, size uint64) {
	stats.CacheSizeBytesGauge(backend).Set(size)
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./pkg/cache/`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add pkg/cache/metrics.go
git commit -m "feat(cache): add metric recording helpers"
```

---

### Task 5: Interface types (`pkg/cache/types.go`)

**Files:**
- Create: `pkg/cache/types.go`

- [ ] **Step 1: Write interfaces**

```go
package cache

import "context"

type StringCache interface {
	Get(ctx context.Context, key Key) (string, bool, error)
	Set(ctx context.Context, key Key, value string, ttl TTL) error
	SetNX(ctx context.Context, key Key, value string, ttl TTL) (bool, error)
	Del(ctx context.Context, key Key) error
	Exists(ctx context.Context, key Key) (bool, error)
	Expire(ctx context.Context, key Key, ttl TTL) error
}

type IntCache interface {
	Get(ctx context.Context, key Key) (int64, error)
	Set(ctx context.Context, key Key, value int64, ttl TTL) error
	Incr(ctx context.Context, key Key) (int64, error)
	IncrWithTTL(ctx context.Context, key Key, ttl TTL) (int64, error)
}

type SetCache interface {
	Add(ctx context.Context, key Key, ttl TTL, members ...string) (int64, error)
	IsMember(ctx context.Context, key Key, member string) (bool, error)
	Members(ctx context.Context, key Key) ([]string, error)
	Remove(ctx context.Context, key Key, members ...string) (int64, error)
	Clear(ctx context.Context, key Key) error
}

type ListCache interface {
	Push(ctx context.Context, key Key, values ...string) (int64, error)
	Range(ctx context.Context, key Key, start, stop int64) ([]string, error)
	Len(ctx context.Context, key Key) (int64, error)
	Clear(ctx context.Context, key Key) error
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./pkg/cache/`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add pkg/cache/types.go
git commit -m "feat(cache): add StringCache, IntCache, SetCache, ListCache interfaces"
```

---

### Task 6: Enhance RistrettoStore (`pkg/cache/cache.go`)

Implement `StringCache` on the existing `Cache` struct with metrics instrumentation.

**Files:**
- Modify: `pkg/cache/cache.go`

- [ ] **Step 1: Add StringCache methods with metrics to Cache struct**

Add to `cache.go` after the existing `Wait()` method:

```go
func (c *Cache) Get(ctx context.Context, key Key) (string, bool, error) {
	val, ok := c.i.Get(key.String())
	if !ok {
		recordMiss("ristretto")
		return "", false, nil
	}
	recordHit("ristretto")
	s, ok := val.(string)
	if !ok {
		return "", false, nil
	}
	return s, true, nil
}

func (c *Cache) Set(ctx context.Context, key Key, value string, ttl TTL) error {
	c.i.SetWithTTL(key.String(), value, 1, ttl.Duration())
	return nil
}

func (c *Cache) SetNX(ctx context.Context, key Key, value string, ttl TTL) (bool, error) {
	_, exists := c.i.Get(key.String())
	if exists {
		return false, nil
	}
	c.i.SetWithTTL(key.String(), value, 1, ttl.Duration())
	return true, nil
}

func (c *Cache) Del(ctx context.Context, key Key) error {
	c.i.Del(key.String())
	recordEviction("ristretto")
	return nil
}

func (c *Cache) Exists(ctx context.Context, key Key) (bool, error) {
	_, ok := c.i.Get(key.String())
	return ok, nil
}

func (c *Cache) Expire(ctx context.Context, key Key, ttl TTL) error {
	val, ok := c.i.Get(key.String())
	if !ok {
		return nil
	}
	c.i.SetWithTTL(key.String(), val, 1, ttl.Duration())
	return nil
}
```

- [ ] **Step 2: Write expanded tests**

Add to `pkg/cache/cache_test.go`:

```go
func TestCacheStringCache(t *testing.T) {
	cache, err := NewCache(&config.Type{})
	require.NoError(t, err)
	require.NotNil(t, cache)
	defer cache.Wait()

	t.Run("Get and Set with Key", func(t *testing.T) {
		key := NewKey("test", "string", "get")
		err := cache.Set(context.Background(), key, "hello", TTLShort)
		require.NoError(t, err)
		cache.Wait()

		val, ok, err := cache.Get(context.Background(), key)
		require.NoError(t, err)
		require.True(t, ok)
		require.Equal(t, "hello", val)
	})

	t.Run("Get miss returns false", func(t *testing.T) {
		key := NewKey("test", "string", "miss")
		val, ok, err := cache.Get(context.Background(), key)
		require.NoError(t, err)
		require.False(t, ok)
		require.Equal(t, "", val)
	})

	t.Run("SetNX first call returns true", func(t *testing.T) {
		key := NewKey("test", "setnx", "first")
		ok, err := cache.SetNX(context.Background(), key, "1", TTLShort)
		require.NoError(t, err)
		require.True(t, ok)
		cache.Wait()
	})

	t.Run("SetNX second call returns false", func(t *testing.T) {
		key := NewKey("test", "setnx", "second")
		_, _ = cache.SetNX(context.Background(), key, "1", TTLShort)
		cache.Wait()
		ok, err := cache.SetNX(context.Background(), key, "1", TTLShort)
		require.NoError(t, err)
		require.False(t, ok)
	})

	t.Run("Exists finds set key", func(t *testing.T) {
		key := NewKey("test", "exists", "yes")
		err := cache.Set(context.Background(), key, "val", TTLShort)
		require.NoError(t, err)
		cache.Wait()
		ok, err := cache.Exists(context.Background(), key)
		require.NoError(t, err)
		require.True(t, ok)
	})

	t.Run("Exists misses unset key", func(t *testing.T) {
		key := NewKey("test", "exists", "no")
		ok, err := cache.Exists(context.Background(), key)
		require.NoError(t, err)
		require.False(t, ok)
	})

	t.Run("Del removes key", func(t *testing.T) {
		key := NewKey("test", "del", "gone")
		err := cache.Set(context.Background(), key, "val", TTLShort)
		require.NoError(t, err)
		cache.Wait()
		err = cache.Del(context.Background(), key)
		require.NoError(t, err)
		cache.Wait()
		ok, err := cache.Exists(context.Background(), key)
		require.NoError(t, err)
		require.False(t, ok)
	})

	t.Run("Expire refreshes TTL", func(t *testing.T) {
		key := NewKey("test", "expire", "refresh")
		err := cache.Set(context.Background(), key, "val", TTLShort)
		require.NoError(t, err)
		cache.Wait()
		err = cache.Expire(context.Background(), key, TTLLong)
		require.NoError(t, err)
		cache.Wait()
		ok, err := cache.Exists(context.Background(), key)
		require.NoError(t, err)
		require.True(t, ok)
	})
}
```

- [ ] **Step 3: Run tests to verify they pass**

Run: `go test ./pkg/cache/ -run TestCacheStringCache -v`
Expected: PASS

- [ ] **Step 4: Run all cache tests to verify no regression**

Run: `go test ./pkg/cache/ -v`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/cache/cache.go pkg/cache/cache_test.go
git commit -m "feat(cache): implement StringCache on RistrettoStore with metrics"
```

---

### Task 7: RedisStore implementation (`pkg/cache/redis.go`)

**Files:**
- Create: `pkg/cache/redis.go`
- Create: `pkg/cache/redis_test.go`

This is the largest single task. RedisStore wraps `rdb.Client` and implements all four interfaces.

- [ ] **Step 1: Write RedisStore struct and constructor**

```go
package cache

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/flowline-io/flowbot/pkg/rdb"
)

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
}
```

- [ ] **Step 2: Implement StringCache methods**

```go
func (s *RedisStore) Get(ctx context.Context, key Key) (string, bool, error) {
	val, err := s.client.Get(ctx, key.String()).Result()
	if err == redis.Nil {
		recordMiss("redis")
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("redis get %s: %w", key.String(), err)
	}
	recordHit("redis")
	return val, true, nil
}

func (s *RedisStore) Set(ctx context.Context, key Key, value string, ttl TTL) error {
	return s.client.Set(ctx, key.String(), value, ttl.Duration()).Err()
}

func (s *RedisStore) SetNX(ctx context.Context, key Key, value string, ttl TTL) (bool, error) {
	return s.client.SetNX(ctx, key.String(), value, ttl.Duration()).Result()
}

func (s *RedisStore) Del(ctx context.Context, key Key) error {
	err := s.client.Del(ctx, key.String()).Err()
	if err != nil {
		return fmt.Errorf("redis del %s: %w", key.String(), err)
	}
	recordEviction("redis")
	return nil
}

func (s *RedisStore) Exists(ctx context.Context, key Key) (bool, error) {
	n, err := s.client.Exists(ctx, key.String()).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists %s: %w", key.String(), err)
	}
	if n > 0 {
		recordHit("redis")
		return true, nil
	}
	recordMiss("redis")
	return false, nil
}

func (s *RedisStore) Expire(ctx context.Context, key Key, ttl TTL) error {
	return s.client.Expire(ctx, key.String(), ttl.Duration()).Err()
}
```

- [ ] **Step 3: Implement IntCache methods**

```go
func (s *RedisStore) GetInt64(ctx context.Context, key Key) (int64, error) {
	val, err := s.client.Get(ctx, key.String()).Int64()
	if err == redis.Nil {
		recordMiss("redis")
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("redis get int64 %s: %w", key.String(), err)
	}
	recordHit("redis")
	return val, nil
}

func (s *RedisStore) SetInt64(ctx context.Context, key Key, value int64, ttl TTL) error {
	return s.client.Set(ctx, key.String(), value, ttl.Duration()).Err()
}

func (s *RedisStore) Incr(ctx context.Context, key Key) (int64, error) {
	return s.client.Incr(ctx, key.String()).Result()
}

func (s *RedisStore) IncrWithTTL(ctx context.Context, key Key, ttl TTL) (int64, error) {
	newCount, err := s.client.Incr(ctx, key.String()).Result()
	if err != nil {
		return 0, fmt.Errorf("redis incr %s: %w", key.String(), err)
	}
	if newCount == 1 {
		if err := s.client.Expire(ctx, key.String(), ttl.Duration()).Err(); err != nil {
			return newCount, fmt.Errorf("redis expire %s: %w", key.String(), err)
		}
	}
	return newCount, nil
}
```

- [ ] **Step 4: Implement SetCache methods**

```go
func (s *RedisStore) Add(ctx context.Context, key Key, ttl TTL, members ...string) (int64, error) {
	n, err := s.client.SAdd(ctx, key.String(), toAny(members)...).Result()
	if err != nil {
		return 0, fmt.Errorf("redis sadd %s: %w", key.String(), err)
	}
	if ttl.Duration() > 0 {
		s.client.Expire(ctx, key.String(), ttl.Duration())
	}
	return n, nil
}

func (s *RedisStore) IsMember(ctx context.Context, key Key, member string) (bool, error) {
	return s.client.SIsMember(ctx, key.String(), member).Result()
}

func (s *RedisStore) Members(ctx context.Context, key Key) ([]string, error) {
	return s.client.SMembers(ctx, key.String()).Result()
}

func (s *RedisStore) Remove(ctx context.Context, key Key, members ...string) (int64, error) {
	return s.client.SRem(ctx, key.String(), toAny(members)...).Result()
}

func (s *RedisStore) Clear(ctx context.Context, key Key) error {
	return s.client.Del(ctx, key.String()).Err()
}

func toAny(ss []string) []any {
	res := make([]any, len(ss))
	for i, s := range ss {
		res[i] = s
	}
	return res
}
```

- [ ] **Step 5: Implement ListCache methods**

```go
func (s *RedisStore) Push(ctx context.Context, key Key, values ...string) (int64, error) {
	args := make([]any, len(values))
	for i, v := range values {
		args[i] = v
	}
	return s.client.RPush(ctx, key.String(), args...).Result()
}

func (s *RedisStore) Range(ctx context.Context, key Key, start, stop int64) ([]string, error) {
	return s.client.LRange(ctx, key.String(), start, stop).Result()
}

func (s *RedisStore) Len(ctx context.Context, key Key) (int64, error) {
	return s.client.LLen(ctx, key.String()).Result()
}
```

`Clear` is already defined above for SetCache — same signature satisfies both interfaces.

- [ ] **Step 6: Add utility methods for scan and raw operations**

```go
func (s *RedisStore) ScanKeys(ctx context.Context, pattern string, count int64) ([]string, error) {
	var keys []string
	var cursor uint64
	for {
		result, nextCursor, err := s.client.Scan(ctx, cursor, pattern, count).Result()
		if err != nil {
			return nil, fmt.Errorf("redis scan %s: %w", pattern, err)
		}
		keys = append(keys, result...)
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return keys, nil
}

func (s *RedisStore) ExistsRaw(ctx context.Context, key string) (bool, error) {
	n, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists %s: %w", key, err)
	}
	return n > 0, nil
}
```

- [ ] **Step 7: Write RedisStore tests**

```go
package cache

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestRedisStoreStringCache(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store := NewRedisStore(client)

	t.Run("Set and Get string", func(t *testing.T) {
		key := NewKey("test", "string", "get")
		err := store.Set(context.Background(), key, "hello", TTLShort)
		require.NoError(t, err)

		val, ok, err := store.Get(context.Background(), key)
		require.NoError(t, err)
		require.True(t, ok)
		require.Equal(t, "hello", val)
	})

	t.Run("Get miss", func(t *testing.T) {
		key := NewKey("test", "string", "nope")
		val, ok, err := store.Get(context.Background(), key)
		require.NoError(t, err)
		require.False(t, ok)
		require.Equal(t, "", val)
	})

	t.Run("SetNX first call returns true", func(t *testing.T) {
		key := NewKey("test", "setnx", "new")
		ok, err := store.SetNX(context.Background(), key, "1", TTLShort)
		require.NoError(t, err)
		require.True(t, ok)
	})

	t.Run("SetNX second call returns false", func(t *testing.T) {
		key := NewKey("test", "setnx", "dup")
		_, _ = store.SetNX(context.Background(), key, "1", TTLShort)
		ok, err := store.SetNX(context.Background(), key, "1", TTLShort)
		require.NoError(t, err)
		require.False(t, ok)
	})

	t.Run("Exists and Del", func(t *testing.T) {
		key := NewKey("test", "del", "temp")
		_ = store.Set(context.Background(), key, "x", TTLShort)
		ok, _ := store.Exists(context.Background(), key)
		require.True(t, ok)
		_ = store.Del(context.Background(), key)
		ok, _ = store.Exists(context.Background(), key)
		require.False(t, ok)
	})

	t.Run("Expire", func(t *testing.T) {
		key := NewKey("test", "expire", "key")
		_ = store.Set(context.Background(), key, "x", TTLMinute)
		_ = store.Expire(context.Background(), key, TTLLong)
		ok, _ := store.Exists(context.Background(), key)
		require.True(t, ok)
	})
}

func TestRedisStoreIntCache(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store := NewRedisStore(client)

	t.Run("SetInt64 and GetInt64", func(t *testing.T) {
		key := NewKey("test", "int", "val")
		err := store.SetInt64(context.Background(), key, 42, TTLShort)
		require.NoError(t, err)
		val, err := store.GetInt64(context.Background(), key)
		require.NoError(t, err)
		require.Equal(t, int64(42), val)
	})

	t.Run("GetInt64 miss returns 0", func(t *testing.T) {
		key := NewKey("test", "int", "nope")
		val, err := store.GetInt64(context.Background(), key)
		require.NoError(t, err)
		require.Equal(t, int64(0), val)
	})

	t.Run("Incr creates counter", func(t *testing.T) {
		key := NewKey("test", "incr", "cnt")
		val, err := store.Incr(context.Background(), key)
		require.NoError(t, err)
		require.Equal(t, int64(1), val)
		val, err = store.Incr(context.Background(), key)
		require.NoError(t, err)
		require.Equal(t, int64(2), val)
	})

	t.Run("IncrWithTTL sets expiry on first call", func(t *testing.T) {
		key := NewKey("test", "incr", "ttl")
		n, err := store.IncrWithTTL(context.Background(), key, TTLMonth)
		require.NoError(t, err)
		require.Equal(t, int64(1), n)
		ok, _ := store.Exists(context.Background(), key)
		require.True(t, ok)

		n, err = store.IncrWithTTL(context.Background(), key, TTLMonth)
		require.NoError(t, err)
		require.Equal(t, int64(2), n)
	})
}

func TestRedisStoreSetCache(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store := NewRedisStore(client)

	t.Run("Add and IsMember", func(t *testing.T) {
		key := NewKey("test", "set", "items")
		n, err := store.Add(context.Background(), key, TTLShort, "a", "b")
		require.NoError(t, err)
		require.Equal(t, int64(2), n)

		ok, err := store.IsMember(context.Background(), key, "a")
		require.NoError(t, err)
		require.True(t, ok)
	})

	t.Run("IsMember false for missing", func(t *testing.T) {
		key := NewKey("test", "set", "missing")
		ok, err := store.IsMember(context.Background(), key, "x")
		require.NoError(t, err)
		require.False(t, ok)
	})

	t.Run("Members returns all", func(t *testing.T) {
		key := NewKey("test", "set", "members")
		_, _ = store.Add(context.Background(), key, TTLShort, "x", "y")
		m, err := store.Members(context.Background(), key)
		require.NoError(t, err)
		require.Len(t, m, 2)
		require.Contains(t, m, "x")
		require.Contains(t, m, "y")
	})

	t.Run("Remove and Clear", func(t *testing.T) {
		key := NewKey("test", "set", "clear")
		_, _ = store.Add(context.Background(), key, TTLShort, "a", "b", "c")
		n, err := store.Remove(context.Background(), key, "a")
		require.NoError(t, err)
		require.Equal(t, int64(1), n)
		_ = store.Clear(context.Background(), key)
		ok, _ := store.Exists(context.Background(), key)
		require.False(t, ok)
	})
}

func TestRedisStoreListCache(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store := NewRedisStore(client)

	t.Run("Push and Range", func(t *testing.T) {
		key := NewKey("test", "list", "items")
		n, err := store.Push(context.Background(), key, "a", "b")
		require.NoError(t, err)
		require.Equal(t, int64(2), n)

		items, err := store.Range(context.Background(), key, 0, -1)
		require.NoError(t, err)
		require.Equal(t, []string{"a", "b"}, items)
	})

	t.Run("Len counts items", func(t *testing.T) {
		key := NewKey("test", "list", "len")
		_, _ = store.Push(context.Background(), key, "a", "b", "c")
		n, err := store.Len(context.Background(), key)
		require.NoError(t, err)
		require.Equal(t, int64(3), n)
	})

	t.Run("Clear empties list", func(t *testing.T) {
		key := NewKey("test", "list", "clear")
		_, _ = store.Push(context.Background(), key, "a")
		_ = store.Clear(context.Background(), key)
		n, _ := store.Len(context.Background(), key)
		require.Equal(t, int64(0), n)
	})
}
```

- [ ] **Step 8: Add miniredis dependency**

Run: `go get github.com/alicebob/miniredis/v2`
Expected: dependency added to go.mod

- [ ] **Step 9: Run RedisStore tests**

Run: `go test ./pkg/cache/ -run "TestRedisStore" -v`
Expected: all PASS

- [ ] **Step 10: Run all cache tests**

Run: `go test ./pkg/cache/ -v`
Expected: all PASS

- [ ] **Step 11: Commit**

```bash
git add pkg/cache/redis.go pkg/cache/redis_test.go go.mod go.sum
git commit -m "feat(cache): add RedisStore implementing StringCache, IntCache, SetCache, ListCache"
```

---

### Task 8: Register RedisStore in DI (`internal/server/fx.go`)

**Files:**
- Modify: `internal/server/fx.go`

- [ ] **Step 1: Add RedisStore provider**

In `internal/server/fx.go`, in the `fx.Provide(` block, add after `rdb.NewClient`:

```go
fx.Provide(
    config.NewConfig,
    cache.NewCache,
    rdb.NewClient,
    cache.NewRedisStore,       // <-- ADD THIS
    search.NewClient,
    event.NewRouter,
    ...
),
```

The final `fx.Provide` block should be:

```go
fx.Provide(
    config.NewConfig,
    cache.NewCache,
    rdb.NewClient,
    cache.NewRedisStore,
    search.NewClient,
    event.NewRouter,
    event.NewSubscriber,
    event.NewPublisher,
    slack.NewDriver,
    trace.NewTracerProvider,
    newController,
    newDatabaseAdapter,
    newHTTPServer,
),
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/server/`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add internal/server/fx.go
git commit -m "feat(server): register RedisStore in dependency injection"
```

---

## Phase 2: Module Migration

### Task 9: Migrate chat + online in `internal/server/func.go`

**Files:**
- Modify: `internal/server/func.go`

Replace `rdb.Client` calls for chat/online with `RedisStore`.

- [ ] **Step 1: Replace chat session code**

In `directIncomingMessage` (currently lines 135-166), replace:

```go
chatKey := fmt.Sprintf("chat:%s", uid)
session, err := rdb.Client.Get(ctx.Context(), chatKey).Result()
if err != nil {
    if !errors.Is(err, redis.Nil) {
        flog.Error(err)
    }
}
```

with:

```go
chatKey := cache.NewKey("chat", "session", uid)
var session string
s, ok, err := cacheInstance.Get(ctx.Context(), chatKey)
if err != nil {
    flog.Error(err)
}
if ok {
    session = s
}
```

Replace:

```go
err = rdb.Client.Set(ctx.Context(), chatKey, types.Id(), 24*time.Hour).Err()
```

with:

```go
err = cacheInstance.Set(ctx.Context(), chatKey, types.Id(), cache.TTLSession)
```

Replace:

```go
err = rdb.Client.Del(ctx.Context(), chatKey).Err()
```

with:

```go
err = cacheInstance.Del(ctx.Context(), chatKey)
```

- [ ] **Step 2: Replace online status code**

In `onlineStatus` (currently lines 341-357), replace:

```go
key := fmt.Sprintf("online:%s", med.UserId)
_, err := rdb.Client.Get(ctx, key).Result()
if errors.Is(err, redis.Nil) {
    rdb.Client.Set(ctx, key, time.Now().Unix(), 30*time.Minute)
} else if err != nil {
    return
} else {
    rdb.Client.Expire(ctx, key, 30*time.Minute)
}
```

with:

```go
key := cache.NewKey("online", "user", med.UserId)
_, ok, _ := cacheInstance.Get(ctx, key)
if !ok {
    cacheInstance.Set(ctx, key, strconv.FormatInt(time.Now().Unix(), 10), cache.TTLSession)
} else {
    cacheInstance.Expire(ctx, key, cache.TTLSession)
}
```

- [ ] **Step 3: Replace agent online code**

In `agentAction` (lines 413-435), replace:

```go
check, err := rdb.Client.Get(ctx.Context(), fmt.Sprintf("online:%s", hostid)).Result()
if err != nil && !errors.Is(err, redis.Nil) {
    return nil, errors.New("get agent online error")
}
if check == "" {
    // ... send online notification ...
}

_, err = rdb.Client.Set(ctx.Context(), fmt.Sprintf("online:%s", hostid), time.Now().Unix(), 2*time.Minute).Result()
```

with:

```go
key := cache.NewKey("online", "agent", hostid)
_, ok, _ := cacheInstance.Get(ctx.Context(), key)
if !ok {
    // ... send online notification ...
}

err = cacheInstance.Set(ctx.Context(), key, strconv.FormatInt(time.Now().Unix(), 10), cache.TTLShort)
```

Replace agent offline:

```go
_, err = rdb.Client.Del(ctx.Context(), fmt.Sprintf("online:%s", hostid)).Result()
```

with:

```go
err = cacheInstance.Del(ctx.Context(), key)
```

- [ ] **Step 4: Add imports to func.go**

The existing import of `"github.com/redis/go-redis/v9"` can be removed.
Add `"github.com/flowline-io/flowbot/pkg/cache"` import.

```
import (
    ...
    cachePkg "github.com/flowline-io/flowbot/pkg/cache"
    ...
)
```

Remove unused `rdb` import.
Remove unused `fmt.Sprintf` for the old online key format.
Add `"strconv"` for `strconv.FormatInt`.

- [ ] **Step 5: Handle global DI pattern**

The file currently uses `rdb.Client` (package-level global). We need `cacheInstance` — check how fx provides the store. Since `func.go` is in `package server`, the fx store would be passed in struct fields.

For now, accept `*cache.RedisStore` as a function parameter via DI structure. In `func.go`, add a package-level variable set during server initialization:

At the top of `func.go` after imports:
```go
var cacheInstance *cache.RedisStore
```

And in the server constructor (where fx wires it):
```go
func initCache(store *cache.RedisStore) {
    cacheInstance = store
}
```

Or the easiest approach: since `func.go` functions are called from `newController` or handlers that already receive DI parameters, pass `*cache.RedisStore` through those callers. The `directIncomingMessage`, `onlineStatus`, and `agentAction` functions receive parameters from the DI-wired controller.

Check `newController` in `internal/server/`. We add `store *cache.RedisStore` as a parameter and pass it to the place that sets `func.go` globals.

If these functions are called as goroutines in event handlers (not directly injected), the simplest approach: store it as package global set during init:

The `handleEvents` `fx.Invoke` function in fx.go can accept `*cache.RedisStore` and set the global:

In `func.go`:
```go
var cacheStore *cache.RedisStore

func SetCacheStore(store *cache.RedisStore) {
    cacheStore = store
}
```

Then in fx.go's `handleEvents` or a new invoke:
```go
func setCacheStore(store *cache.RedisStore) {
    server.SetCacheStore(store)
}
```

Simplest approach: Use an fx Invoke function in `internal/server/`:

```go
func setServerCache(store *cache.RedisStore) {
    // sets package-level global in internal/server
    cacheStore = store
}
```

Add to fx:
```go
fx.Invoke(
    setServerCache,
    handleRoutes,
    ...
)
```

For now, implement the direct approach: add `SetCacheStore` and wire it.

- [ ] **Step 6: Verify compilation**

Run: `go build ./internal/server/`
Expected: no errors

- [ ] **Step 7: Run server tests**

Run: `go test ./internal/server/ -v`
Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add internal/server/func.go internal/server/fx.go
git commit -m "refactor(server): migrate chat and online to RedisStore"
```

---

### Task 10: Migrate online count cron (`internal/modules/server/cron.go`)

**Files:**
- Modify: `internal/modules/server/cron.go`

- [ ] **Step 1: Replace online count logic**

In the `server_user_online_change` cron action (currently lines 29-36):

Current:
```go
keys, _ := rdb.Client.Keys(ctx.Context(), "online:*").Result()
currentCount := int64(len(keys))
lastKey := fmt.Sprintf("server:cron:online_count_last:%s", ctx.AsUser.String())
lastCount, _ := rdb.Client.Get(ctx.Context(), lastKey).Int64()
rdb.Client.Set(ctx.Context(), lastKey, currentCount, redis.KeepTTL)
```

Replace with:
```go
keys, _ := rdb.Client.Keys(ctx.Context(), "online:*").Result()
currentCount := int64(len(keys))
lastKey := cache.NewKey("online", "cron_count", ctx.AsUser.String())
lastCount, err := cacheStore.GetInt64(ctx.Context(), lastKey)
if err != nil {
    flog.Error(err)
}
_ = cacheStore.SetInt64(ctx.Context(), lastKey, currentCount, cache.TTLMonth)
```

- [ ] **Step 2: Remove unused imports**

Remove `"github.com/redis/go-redis/v9"` import.
Remove `"fmt"` import (no longer needed for `fmt.Sprintf`).

- [ ] **Step 3: Add cache package access**

The module server cron uses `package server` inside `internal/modules/server/`.
Add a package-level variable `cacheStore *cache.RedisStore` set via the same DI pattern.

Since `internal/modules/server/` already has a `newController` style entry point, search how it gets DI.

Add to `internal/modules/server/cron.go` at top:
```go
var cacheStore *cache.RedisStore

func SetCacheStore(store *cache.RedisStore) {
    cacheStore = store
}
```

Wire in `internal/server/fx.go`:
```go
func initModuleCacheStore(store *cache.RedisStore) {
    serverModule.SetCacheStore(store)
}
```

Or, if the module init path already passes through fx, inject it through the module struct. Check how `server` module is initialized:

Looking at `internal/modules/server/`, there should be a module.go file. Let's check.

- [ ] **Step 4: Verify compilation**

Run: `go build ./internal/...`
Expected: no errors

- [ ] **Step 5: Run tests**

Run: `go test ./internal/modules/server/ -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/modules/server/cron.go
git commit -m "refactor(server/cron): migrate online count to RedisStore IntCache"
```

---

### Task 11: Migrate crawler (`pkg/crawler/crawler.go`)

**Files:**
- Modify: `pkg/crawler/crawler.go`

- [ ] **Step 1: Add RedisStore field to Crawler struct**

```go
type Crawler struct {
    jobs    map[string]Rule
    outCh   chan Result
    stop    chan struct{}
    store   *cache.RedisStore
    Send    func(id, name string, out []map[string]string)
}

func New(store *cache.RedisStore) *Crawler {
    return &Crawler{
        jobs:  make(map[string]Rule),
        outCh: make(chan Result, 10),
        stop:  make(chan struct{}),
        store: store,
    }
}
```

- [ ] **Step 2: Rewrite filter method**

Replace the entire filter method body. The key logic changes:

```go
func (s *Crawler) filter(name, mode string, latest []map[string]string) []map[string]string {
    ctx := context.Background()
    sentKey := cache.NewKey("crawler", "sent", name)
    todoKey := cache.NewKey("crawler", "todo", name)
    sendTimeKey := cache.NewKey("crawler", "sendtime", name)

    // sent — read existing items from set
    oldArr, _ := s.store.Members(ctx, sentKey)
    var old []map[string]string
    for _, item := range oldArr {
        var tmp map[string]string
        _ = sonic.Unmarshal([]byte(item), &tmp)
        if tmp != nil {
            old = append(old, tmp)
        }
    }

    // todo — read pending daily batch
    todoArr, _ := s.store.Members(ctx, todoKey)
    var todo []map[string]string
    for _, item := range todoArr {
        var tmp map[string]string
        _ = sonic.Unmarshal([]byte(item), &tmp)
        if tmp != nil {
            todo = append(todo, tmp)
        }
    }

    old = append(old, todo...)
    diff := stringSliceDiff(latest, old)

    switch mode {
    case "instant":
        _ = s.store.Set(ctx, sendTimeKey, strconv.FormatInt(time.Now().Unix(), 10), cache.TTLMedium)
    case "daily":
        sendStr, _, _ := s.store.Get(ctx, sendTimeKey)
        oldSend := int64(0)
        if sendStr != "" {
            oldSend, _ = strconv.ParseInt(sendStr, 10, 64)
        }

        if time.Now().Unix()-oldSend < 24*60*60 {
            for _, item := range diff {
                d, _ := sonic.Marshal(item)
                _, _ = s.store.Add(ctx, todoKey, cache.TTLDay, string(d))
            }
            return nil
        }

        diff = append(diff, todo...)
        _ = s.store.Set(ctx, sendTimeKey, strconv.FormatInt(time.Now().Unix(), 10), cache.TTLMedium)
    default:
        return nil
    }

    // add data to sent set with 30-day TTL
    for _, item := range diff {
        d, _ := sonic.Marshal(item)
        _, _ = s.store.Add(ctx, sentKey, cache.TTLMonth, string(d))
    }

    // clear to-do list
    _ = s.store.Clear(ctx, todoKey)

    return diff
}
```

- [ ] **Step 3: Remove old imports**

Remove `"github.com/redis/go-redis/v9"` import.
Remove `"github.com/flowline-io/flowbot/pkg/rdb"` import.
Add `"github.com/flowline-io/flowbot/pkg/cache"` import.

- [ ] **Step 4: Update callers of `crawler.New()`**

Search for `crawler.New()` call sites and pass a `*cache.RedisStore`.
Check `internal/server/fx.go` or wherever crawler is created.

- [ ] **Step 5: Verify compilation**

Run: `go build ./pkg/crawler/ && go build ./internal/...`
Expected: no errors

- [ ] **Step 6: Commit**

```bash
git add pkg/crawler/crawler.go
git commit -m "refactor(crawler): migrate to RedisStore SetCache with TTLMonth"
```

---

### Task 12: Migrate cron filter (`pkg/types/ruleset/cron/cron.go`)

**Files:**
- Modify: `pkg/types/ruleset/cron/cron.go`

- [ ] **Step 1: Add RedisStore to Ruleset struct**

```go
type Ruleset struct {
    stop      chan struct{}
    Type      string
    outCh     chan result
    cronRules []Rule
    store     *cache.RedisStore
}
```

Update constructor:

```go
func NewCronRuleset(name string, rules []Rule, store *cache.RedisStore) *Ruleset {
    return &Ruleset{
        stop:      make(chan struct{}),
        Type:      name,
        outCh:     make(chan result, 100),
        cronRules: rules,
        store:     store,
    }
}
```

- [ ] **Step 2: Replace filter method**

In the `filter` method (lines 190-208), replace:

```go
func (r *Ruleset) filter(res result) result {
    filterKey := fmt.Sprintf("cron:%s:%s:filter", res.name, res.ctx.AsUser)

    d := un(res.payload)
    s := sha1.New()
    _, _ = s.Write(d)
    hash := s.Sum(nil)

    ctx := context.Background()
    state := rdb.Client.SIsMember(ctx, filterKey, hash).Val()
    if state {
        return result{}
    }

    _ = rdb.Client.SAdd(ctx, filterKey, hash)
    return res
}
```

with:

```go
func (r *Ruleset) filter(res result) result {
    key := cache.NewKey("cron", "filter", res.name+":"+string(res.ctx.AsUser))

    d := un(res.payload)
    s := sha1.New()
    _, _ = s.Write(d)
    hash := string(s.Sum(nil))

    ctx := context.Background()
    ok, _ := r.store.IsMember(ctx, key, hash)
    if ok {
        return result{}
    }

    _, _ = r.store.Add(ctx, key, cache.TTLMonth, hash)
    return res
}
```

- [ ] **Step 3: Remove old imports**

Remove `"github.com/flowline-io/flowbot/pkg/rdb"` import.
Remove `"fmt"` import.
Add `"github.com/flowline-io/flowbot/pkg/cache"` import.

- [ ] **Step 4: Update callers of NewCronRuleset**

Search for `cron.NewCronRuleset` call sites and pass the `*cache.RedisStore`.
Check in module files under `internal/modules/`.

- [ ] **Step 5: Verify compilation**

Run: `go build ./pkg/types/ruleset/cron/ && go build ./internal/...`
Expected: no errors

- [ ] **Step 6: Commit**

```bash
git add pkg/types/ruleset/cron/cron.go
git commit -m "refactor(cron/ruleset): migrate filter set to RedisStore SetCache with TTLMonth"
```

---

### Task 13: Migrate rdb metrics (`pkg/rdb/metrics.go`)

**Files:**
- Modify: `pkg/rdb/metrics.go` — deprecate, keep forwarding shim for now

- [ ] **Step 1: Add deprecation comments**

The `SetMetricsInt64` and `GetMetricsInt64` functions in `pkg/rdb/metrics.go` are used by 8+ cron jobs. Rather than rewrite all at once, deprecate the old functions and add new equivalents on `RedisStore`.

Add to `pkg/cache/redis.go`:

```go
func (s *RedisStore) SetMetricsInt64(key string, value int64) {
    k := NewKey("metrics", "gauge", key)
    _ = s.SetInt64(context.Background(), k, value, TTLMonth)
}

func (s *RedisStore) GetMetricsInt64(key string) int64 {
    k := NewKey("metrics", "gauge", key)
    v, _ := s.GetInt64(context.Background(), k)
    return v
}
```

- [ ] **Step 2: Mark old functions deprecated**

Add comment to `pkg/rdb/metrics.go`:

```go
// Deprecated: Use cache.RedisStore.SetMetricsInt64 / GetMetricsInt64 instead.
```

- [ ] **Step 3: Update callers**

Search for `rdb.SetMetricsInt64` and `rdb.GetMetricsInt64` call sites.
Replace with `store.SetMetricsInt64` / `store.GetMetricsInt64` where the store is available.

For cron modules that don't have a store yet (they call rdb directly), keep using the deprecated functions for now and migrate in a follow-up.

Caller files to update:
- `internal/modules/server/module.go` (or wherever ModuleTotalCounter/ModuleRunTotalCounter is used)
- `pkg/rdb/metrics` consumers: search with grep
- `internal/modules/server/cron.go` — docker/monitor counts already in the module

- [ ] **Step 4: Verify compilation**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add pkg/rdb/metrics.go pkg/cache/redis.go
git commit -m "refactor(rdb/metrics): deprecate in favor of RedisStore.SetMetricsInt64"
```

---

### Task 14: Migrate bloom filter (`pkg/rdb/unique.go`)

**Files:**
- Modify: `pkg/rdb/unique.go` — deprecate, add TTL to existing keys

Bloom filter stays with Redis 8.0 native commands. Add TTL only.

- [ ] **Step 1: Add EXPIRE after BFReserve in rdb/unique.go**

In `BloomUnique`:

```go
func BloomUnique(ctx context.Context, id string, latest []any) ([]any, error) {
    result := make([]any, 0)
    uniqueKey := fmt.Sprintf("cache:dedup:%s", id)
    Client.BFReserve(ctx, uniqueKey, 0.001, 1000000)
    // Set 30-day TTL to prevent unbounded growth
    Client.Expire(ctx, uniqueKey, 30*24*time.Hour)

    ...
}
```

Same for `BloomUniqueString`:

```go
func BloomUniqueString(ctx context.Context, id string, latest string) (bool, error) {
    uniqueKey := fmt.Sprintf("cache:dedup:%s", id)
    Client.BFReserve(ctx, uniqueKey, 0.001, 1000000)
    Client.Expire(ctx, uniqueKey, 30*24*time.Hour)
    ...
}
```

- [ ] **Step 2: Update key prefix from `bloom:unique:` to `cache:dedup:`**

Replace `fmt.Sprintf("bloom:unique:%s", id)` with `fmt.Sprintf("cache:dedup:%s", id)` in both functions.

- [ ] **Step 3: Add deprecation comment**

```go
// Deprecated: Use cache.RedisStore for new code. These bloom helpers will be removed in Phase 3 cleanup.
```

- [ ] **Step 4: Verify compilation**

Run: `go build ./pkg/rdb/`
Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add pkg/rdb/unique.go
git commit -m "fix(rdb/bloom): add TTLMonth to bloom keys, rename prefix to cache:dedup"
```

---

### Task 15: Migrate notify rules (`pkg/notify/rules/`)

**Files:**
- Modify: `pkg/notify/rules/engine.go`
- Modify: `pkg/notify/rules/throttle.go`
- Modify: `pkg/notify/rules/aggregate.go`

- [ ] **Step 1: Replace `*redis.Client` with `*cache.RedisStore` in Engine**

In `engine.go`:

```go
type Engine struct {
    mu    sync.RWMutex
    rules []config.NotifyRule
    store *cache.RedisStore
}
```

Update `Init` function to accept `*cache.RedisStore` instead of `*redis.Client`:

```go
func Init(ctx context.Context, c *config.Type, store *cache.RedisStore) *Engine {
    ...
    engine := &Engine{
        store: store,
    }
    ...
}
```

- [ ] **Step 2: Rewrite throttle.go**

Replace `e.redis.Incr` / `e.redis.Expire` / `e.redis.Del` with `e.store.IncrWithTTL` / `e.store.Del`:

```go
func (e *Engine) CheckThrottle(ctx context.Context, ruleID, eventType, channel string, window time.Duration, limit int) (bool, error) {
    if e.store == nil {
        return true, nil
    }

    key := cache.NewKey("notify", "throttle", ruleID+":"+eventType+":"+channel)

    newCount, err := e.store.IncrWithTTL(ctx, key, cache.TTL(window))
    if err != nil {
        flog.Warn("[notify-rules] throttle incr error: %v", err)
        return true, nil
    }

    return newCount <= int64(limit), nil
}

func (e *Engine) ClearThrottle(ctx context.Context, ruleID, eventType, channel string) {
    if e.store == nil {
        return
    }
    key := cache.NewKey("notify", "throttle", ruleID+":"+eventType+":"+channel)
    if err := e.store.Del(ctx, key); err != nil {
        flog.Warn("[notify-rules] throttle clear error: %v", err)
    }
}
```

Note: `IncrWithTTL` combines `INCR` + conditional `EXPIRE`, matching the original TOCTOU-safe pattern. But the original sets EXPIRE only on newCount==1; our `IncrWithTTL` does the same.

Remove unused `Window` parameter in `CheckThrottle` — it was used for `e.redis.Expire(ctx, key, window)`. Now it's passed to `IncrWithTTL` as TTL.

Or better: keep `window` as parameter but use it for TTL:

Actually, looking at the original code: `window` is already the time.Duration. So pass it to `IncrWithTTL`:

```go
newCount, err := e.store.IncrWithTTL(ctx, key, cache.TTL(window))
```

- [ ] **Step 3: Rewrite aggregate.go**

Replace Redis commands:

```go
func (e *Engine) EnqueueForAggregation(ctx context.Context, ruleID, eventType, channel string, payload map[string]any) error {
    if e.store == nil {
        return nil
    }

    key := cache.NewKey("notify", "agg:buffer", ruleID+":"+eventType+":"+channel)

    data, err := sonic.Marshal(payload)
    if err != nil {
        return fmt.Errorf("failed to marshal aggregate payload: %w", err)
    }

    if _, err := e.store.Push(ctx, key, string(data)); err != nil {
        return fmt.Errorf("failed to push to aggregate list: %w", err)
    }

    return nil
}

func (e *Engine) SetAggregateTimer(ctx context.Context, ruleID, eventType, channel string, window time.Duration) (bool, error) {
    if e.store == nil {
        return false, nil
    }

    key := cache.NewKey("notify", "agg:timer", ruleID+":"+eventType+":"+channel)

    ok, err := e.store.SetNX(ctx, key, "1", cache.TTL(window))
    if err != nil {
        return false, fmt.Errorf("failed to set aggregate timer: %w", err)
    }

    return ok, nil
}

func (e *Engine) FlushAggregation(ctx context.Context, ruleID, eventType, channel string) ([]map[string]any, error) {
    if e.store == nil {
        return nil, nil
    }

    key := cache.NewKey("notify", "agg:buffer", ruleID+":"+eventType+":"+channel)

    items, err := e.store.Range(ctx, key, 0, -1)
    if err != nil {
        return nil, fmt.Errorf("failed to read aggregate list: %w", err)
    }

    if len(items) == 0 {
        return nil, nil
    }

    var payloads []map[string]any
    for _, item := range items {
        var payload map[string]any
        if err := sonic.Unmarshal([]byte(item), &payload); err != nil {
            flog.Warn("[notify-rules] failed to unmarshal aggregate payload: %v", err)
            continue
        }
        payloads = append(payloads, payload)
    }

    if err := e.store.Clear(ctx, key); err != nil {
        flog.Warn("[notify-rules] failed to delete aggregate list: %v", err)
    }

    return payloads, nil
}
```

`ScanExpiredAggregates` stays the same (uses `e.redis.Scan` → `e.store.client.Scan`). Since `RedisStore.client` is unexported, we need to either:
- Export a `ScanKeys` method on RedisStore, or
- Keep `*redis.Client` in Engine alongside `*cache.RedisStore`

Better: add a `ScanKeys` method to RedisStore:

In `redis.go` add:
```go
func (s *RedisStore) ScanKeys(ctx context.Context, pattern string, count int64) ([]string, error) {
    var keys []string
    var cursor uint64
    for {
        result, nextCursor, err := s.client.Scan(ctx, cursor, pattern, count).Result()
        if err != nil {
            return nil, fmt.Errorf("redis scan %s: %w", pattern, err)
        }
        keys = append(keys, result...)
        cursor = nextCursor
        if cursor == 0 {
            break
        }
    }
    return keys, nil
}
```

Then update `ScanExpiredAggregates` to use this.

- [ ] **Step 4: Update imports**

Remove `"github.com/redis/go-redis/v9"` from throttle.go and aggregate.go.
Add `"github.com/flowline-io/flowbot/pkg/cache"`.
Remove `"fmt"` if no longer needed (key building moved to Key type).
Remove `"strings"` from aggregate.go (no longer needed for key parsing if we add a ParseKey helper).

- [ ] **Step 5: Update engine.go callers**

Search for `rules.Init(ctx, config, rdbClient)` and replace with `rules.Init(ctx, config, cacheStore)`.

- [ ] **Step 6: Update ScanExpiredAggregates**

Replace the scan logic with:

```go
func (e *Engine) ScanExpiredAggregates(ctx context.Context) ([]AggregateKey, error) {
	if e.store == nil {
		return nil, nil
	}

	var keys []AggregateKey
	results, err := e.store.ScanKeys(ctx, "notify:agg:timer:*", 100)
	if err != nil {
		return nil, fmt.Errorf("failed to scan aggregate timers: %w", err)
	}

	for _, key := range results {
		exists, err := e.store.ExistsRaw(ctx, key)
		if err != nil {
			continue
		}
		if !exists {
			if aggKey, ok := parseAggregateKey(key); ok {
				keys = append(keys, aggKey)
			}
		}
	}

	return keys, nil
}
```

- [ ] **Step 7: Verify compilation and tests**

Run: `go build ./pkg/notify/rules/ && go test ./pkg/notify/rules/ -v`
Expected: no build errors, tests pass

Since notify rules may have test files, check: `ls pkg/notify/rules/*_test.go`

- [ ] **Step 8: Commit**

```bash
git add pkg/notify/rules/ pkg/cache/redis.go
git commit -m "refactor(notify/rules): migrate throttle and aggregate to RedisStore"
```

---

### Task 16: Migrate alarm (`pkg/alarm/alarm.go`)

**Files:**
- Modify: `pkg/alarm/alarm.go`

- [ ] **Step 1: Replace cache.aside with StringCache.SetNX**

Replace the `nx` function (lines 71-88):

Current:
```go
func nx(text string) (bool, error) {
    h := sha1.New()
    _, _ = h.Write([]byte(text))
    hash := hex.EncodeToString(h.Sum(nil))
    key := fmt.Sprintf("alarm:%s", hash)

    _, ok := cache.Instance.Get(key)
    if ok {
        return false, nil
    }

    ok = cache.Instance.SetWithTTL(key, "1", 0, 24*time.Hour)
    if !ok {
        return false, nil
    }

    return true, nil
}
```

Replace with:
```go
func nx(text string) (bool, error) {
    h := sha1.New()
    _, _ = h.Write([]byte(text))
    hash := hex.EncodeToString(h.Sum(nil))
    key := cache.NewKey("alarm", "dedup", hash)

    ctx := context.Background()
    ok, err := cache.Instance.SetNX(ctx, key, "1", cache.TTLDay)
    if err != nil {
        return false, err
    }

    return ok, nil
}
```

Note: `SetNX` is atomic (GET + conditional SET), eliminating the TOCTOU race present in the original code.

- [ ] **Step 2: Update imports**

Remove unused `"time"` import (24*time.Hour replaced by cache.TTLDay).
Add `"context"` import.
Add `"github.com/flowline-io/flowbot/pkg/cache"` import.
Remove `"fmt"` import (no longer needed for `fmt.Sprintf("alarm:%s", hash)`).

- [ ] **Step 3: Verify compilation**

Run: `go build ./pkg/alarm/`
Expected: no errors

- [ ] **Step 4: Run alarm tests if any**

Check: `ls pkg/alarm/*_test.go`

- [ ] **Step 5: Commit**

```bash
git add pkg/alarm/alarm.go
git commit -m "refactor(alarm): migrate dedup to RistrettoStore SetNX with TTLDay"
```

---

## Phase 3: Cleanup

### Task 17: Remove deprecated code

**Files:**
- Delete: `pkg/rdb/metrics.go`
- Delete: `pkg/rdb/unique.go`

- [ ] **Step 1: Verify no remaining callers**

Run: `rg "rdb\.(SetMetricsInt64|GetMetricsInt64|BloomUnique|BloomUniqueString)" --include "*.go" internal/ pkg/`
Expected: zero results

- [ ] **Step 2: Verify all tests pass**

Run: `go test ./...`
Expected: all PASS

- [ ] **Step 3: Verify full build**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 4: Delete deprecated files**

```bash
rm pkg/rdb/metrics.go pkg/rdb/unique.go
```

- [ ] **Step 5: Verify build after deletion**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 6: Run lint**

Run: `go tool task lint`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git rm pkg/rdb/metrics.go pkg/rdb/unique.go
git commit -m "refactor(rdb): remove deprecated metrics and bloom helpers"
```

---

## Execution Handoff

After all 17 tasks complete:
1. Run `go tool task lint` to verify code style
2. Run `go tool task test` to verify unit tests
3. Run `go tool task test:specs` for BDD acceptance tests
