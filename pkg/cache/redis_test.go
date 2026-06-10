package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// TestRedisStoreStringCache tests the StringCache interface via RedisStore.
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
		require.Empty(t, val)
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

// TestRedisStoreIntCache tests the IntCache interface via RedisStore.
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

// TestRedisStoreSetCache tests the SetCache interface via RedisStore.
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

// TestRedisStoreListCache tests the ListCache interface via RedisStore.
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

// TestRedisStoreUtilityMethods tests ScanKeys and ExistsRaw utility methods.
func TestRedisStoreUtilityMethods(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store := NewRedisStore(client)

	t.Run("ScanKeys finds matching keys", func(t *testing.T) {
		ctx := context.Background()
		_ = store.Set(ctx, NewKey("scan", "a", "1"), "v", TTLShort)
		_ = store.Set(ctx, NewKey("scan", "a", "2"), "v", TTLShort)
		_ = store.Set(ctx, NewKey("scan", "b", "1"), "v", TTLShort)

		keys, err := store.ScanKeys(ctx, "scan:a:*", 10)
		require.NoError(t, err)
		require.Len(t, keys, 2)
	})

	t.Run("ScanKeys returns empty for no match", func(t *testing.T) {
		keys, err := store.ScanKeys(context.Background(), "nonexistent:*", 10)
		require.NoError(t, err)
		require.Empty(t, keys)
	})

	t.Run("ExistsRaw finds existing key", func(t *testing.T) {
		ctx := context.Background()
		key := NewKey("test", "existsraw", "yes")
		_ = store.Set(ctx, key, "v", TTLShort)

		ok, err := store.ExistsRaw(ctx, key.String())
		require.NoError(t, err)
		require.True(t, ok)
	})

	t.Run("ExistsRaw returns false for missing key", func(t *testing.T) {
		ok, err := store.ExistsRaw(context.Background(), "no:such:key")
		require.NoError(t, err)
		require.False(t, ok)
	})
}

// TestRedisStore_MetricsInt64 tests SetMetricsInt64 and GetMetricsInt64.
func TestRedisStore_MetricsInt64(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store := NewRedisStore(client)

	t.Run("set and get metric", func(t *testing.T) {
		store.SetMetricsInt64("active_connections", 42)
		v := store.GetMetricsInt64("active_connections")
		require.Equal(t, int64(42), v)
	})

	t.Run("get missing metric returns 0", func(t *testing.T) {
		v := store.GetMetricsInt64("nonexistent")
		require.Equal(t, int64(0), v)
	})

	t.Run("overwrite metric", func(t *testing.T) {
		store.SetMetricsInt64("requests", 100)
		store.SetMetricsInt64("requests", 200)
		v := store.GetMetricsInt64("requests")
		require.Equal(t, int64(200), v)
	})
}

// TestRedisStore_Ping tests the Ping method.
func TestRedisStore_Ping(t *testing.T) {
	t.Run("ping succeeds with healthy redis", func(t *testing.T) {
		mr := miniredis.RunT(t)
		client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		t.Cleanup(func() { _ = client.Close() })
		store := NewRedisStore(client)

		dur, err := store.Ping(context.Background())
		require.NoError(t, err)
		require.GreaterOrEqual(t, dur, time.Duration(0))
	})

	t.Run("ping fails with closed client", func(t *testing.T) {
		mr := miniredis.RunT(t)
		client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		store := NewRedisStore(client)
		require.NoError(t, client.Close())

		_, err := store.Ping(context.Background())
		require.Error(t, err)
	})

	t.Run("ping fails after miniredis shutdown", func(t *testing.T) {
		mr := miniredis.RunT(t)
		client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		t.Cleanup(func() { _ = client.Close() })
		store := NewRedisStore(client)

		mr.Close()

		_, err := store.Ping(context.Background())
		require.Error(t, err)
	})

	t.Run("ping nil client panics", func(t *testing.T) {
		store := &RedisStore{}
		require.Panics(t, func() {
			_, _ = store.Ping(context.Background())
		})
	})
}
