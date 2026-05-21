# Ability Invoke Response Caching

Date: 2026-05-21
Status: design-approved

## Problem Statement

Ristretto in-process cache is allocated 1GB (10M counters) but only used by alarm deduplication (`pkg/alarm/alarm.go:77` — `SetNX` with 24h TTL). The `ability.Invoke()` path has no caching layer, causing redundant external API calls:

1. **Bookmark cron redundancy** — Five cron jobs (`bookmarks_tag`, `bookmarks_metrics`, `bookmarks_search`, `bookmarks_task`, `bookmarks_tag_merge`) independently call `ability.Invoke(ctx, hub.CapBookmark, ability.OpBookmarkList, map[string]any{})`, hitting the Karakeep API up to 3+ times within 60 seconds.
2. **Reader/kanban cron redundancy** — Similar patterns: `reader_metrics` and `reader_daily_summary` both call `OpReaderListEntries`; `kanban_metrics` calls `OpKanbanListTasks` every minute.
3. **No read-through caching** — Every `ability.Invoke()` traverses the full chain (Registry → Invoker → Adapter → Provider → external API). Most Read operations fetch identical data across redundant calls.

## Success Criterion

Cache `ability.Invoke()` results for Read operations in Ristretto with 2-minute TTL. Mutation operations invalidate capability-level cached entries on write. Caching is transparent to callers — miss paths degrade to existing behavior. Cache failures never impact correctness.

## Environment

- Go 1.26+
- Ristretto v2 in-process cache (already initialized: 1GB MaxCost, 10M counters)
- Existing: `pkg/cache/cache.go` (global `Instance`), `pkg/ability/invoke.go` (Registry dispatch)

---

## Architecture

### Invocation Flow (with cache)

```
ability.Invoke(ctx, capType, op, params)
  → 1. key := "ability:{capType}:{op}:<sha1(sortedParamsJSON)>"
  → 2. if isMutation(op): cache.DelByPrefix("ability:{capType}:")
  → 3. if !isMutation(op) && !hasCursor(params):
        val, ok := cache.GetRaw(key)
        if ok → sonic.Unmarshal(val, &result) → return result (cache hit)
  → 4. result, err := invoker(ctx, params)  // original path
  → 5. if err != nil: return nil, err        // errors not cached
  → 6. if !isMutation(op) && !hasCursor(params):
        data, _ := sonic.MarshalString(result)
        cache.SetWithTTL(key, data, 1, TTLShort)  // 2 minutes
  → 7. return result
```

### Modified Files

| File | Change | Lines |
|------|--------|-------|
| `pkg/ability/invoke.go` | Cache-aside logic + isMutation helper | ~50 |
| `pkg/cache/cache.go` | `GetRaw([]byte)` / `SetWithTTL([]byte)` overloads + `DelByPrefix` + key index | ~60 |
| `pkg/ability/operations.go` | `IsMutation(op string) bool` | ~15 |

### Unchanged

- Descriptor definitions, Adapter implementations, Provider clients
- RedisStore paths
- Hub registry

---

## Cache Key Design

**Format:** `ability:{capType}:{op}:{sha1(sortedParamsJSON)}`

**Example:** `ability:bookmark:list:9f86d081884c7d659a2feaa`

**Params serialization:** Sort keys alphabetically, then `sonic.MarshalString`. SHA1 digest for a fixed-length, deterministic key (40 hex chars). SHA1 chosen because it is already used in the codebase (`pkg/alarm/alarm.go` for dedup hashing).

**Non-cacheable requests:**
- Params containing a `cursor` field (pagination cursors change per page, near-zero reuse)
- Explicit `cache: false` in params (future manual override)

---

## Operation Type Classification

`IsMutation(op string) bool` uses name-based matching against known mutation verbs:

```go
var mutationVerbs = []string{
    "create", "delete", "update", "move",
    "archive", "attach", "detach", "complete",
    "mark_read", "mark_unread", "star", "unstar",
    "send", "add",
}
```

- **Read (cached):** `list`, `get`, `search`, `check_url`, `list_feeds`, `list_entries`, `list_tasks`, `get_task`, `get_columns`, `search_tasks`
- **Mutation (not cached, invalidates):** everything matching the verbs above

**Safety:** Default is to NOT cache. Only operations that do not match any mutation verb are cached. New operations are safe until explicitly classified.

---

## TTL and Invalidation

**TTL:** `TTLShort` = 2 minutes (already defined in `pkg/cache/ttl.go`)

**Two-tier invalidation:**
1. **TTL auto-expiry** — Ristretto evicts entries after 2 minutes. This is the primary expiration mechanism.
2. **Active invalidation on write** — When `isMutation(op)` is true, `DelByPrefix("ability:{capType}:")` clears all cached entries for that capability.

### Key Index for Prefix Deletion

Ristretto has no native prefix deletion. A lightweight key index is maintained:

```
Cache.keyIndex: sync.Map  // "bookmark" → map[string]struct{} (key set)
```

- `SetWithTTL` registers `key → capType` in the index
- `DelByPrefix(capType)` loads the key set, calls `Del()` on each, then deletes the index entry
- Index write failures are ignored (does not block Set; only affects invalidation completeness)

---

## Serialization

- **Marshal:** `sonic.MarshalString(result)` → `[]byte`
- **Unmarshal:** `sonic.UnmarshalString(data, &result)` → `*InvokeResult`
- **Cached fields:** `Data`, `Page`, `Text`, `Meta`, `Capability`, `Operation`
- **Not cached:** `Events` — events are emitted per-invocation, not replayed from cache

---

## Error Handling

| Scenario | Behavior |
|----------|----------|
| Invoker returns error | Not cached; error propagated to caller |
| sonic marshal fails | Log warning, skip cache, return result normally |
| sonic unmarshal fails | Log warning, treat as cache miss, execute invoker |
| Ristretto Set fails (capacity full) | Ignored; Set returns false, no impact on flow |
| Ristretto Get misses or errors | Treated as cache miss; original invoker runs |
| Key index write fails | Ignored; only affects future DelByPrefix completeness |
| `isMutation` false negative | Read operation not cached (safe, just suboptimal) |
| `isMutation` false positive | Write invalidates cache unnecessarily (safe, just suboptimal) |

**Core principle:** The cache layer is a performance optimization. Any failure in the caching path must never affect correctness. Callers always receive the correct result.

---

## Cost Model

All cached entries use `cost = 1`. With 1GB MaxCost and Ristretto's sample-based eviction (not exact LRU), the effective capacity is millions of entries. Given:
- ~20 distinct cached operation patterns (list, get, search across bookmark/reader/kanban)
- Each with a small number of param variations
- Total cached entries at steady state << 1000

Memory utilization from ~0 to a few MB. Remaining 1GB capacity remains available.

---

## Testing Strategy

### Unit Tests (`pkg/ability/invoke_test.go`)

| Test | Description |
|------|-------------|
| `cache hit returns stored result` | Second call with same params returns cached result without invoking handler |
| `cache miss invokes handler` | First call or expired TTL invokes the handler normally |
| `mutation operation invalidates prefix` | Write op clears all cached entries for that capability |
| `mutation operation result not cached` | Write op result is never stored in cache |
| `different params produce different keys` | Varying params generate distinct cache keys |
| `handler error not cached` | Error from invoker is not stored in cache |
| `cursor param skips cache` | Params with `cursor` bypass cache entirely |

### Unit Tests (`pkg/cache/cache_test.go`)

| Test | Description |
|------|-------------|
| `DelByPrefix removes all matching keys` | Prefix deletion clears indexed keys |
| `DelByPrefix on empty prefix is no-op` | No panic on missing prefix |
| `SetWithTTL registers in key index` | Key index updated on successful Set |
| `GetRaw returns stored bytes` | Raw byte retrieval with type assertion |

---

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Stale data for 2 minutes after write | TTL is short (2 min); homelab single-user scenario can tolerate brief inconsistency |
| Key index memory growth | Index cleaned on `DelByPrefix`; values expire naturally with TTL eviction (index entries linger but are tiny — ~80 bytes per entry) |
| SHA1 collision across params | Non-security use; collision probability ~1/2^80, negligible for cache keys |
| `sonic` edge cases with complex types | If marshal fails, cache is skipped gracefully; no data corruption possible |
