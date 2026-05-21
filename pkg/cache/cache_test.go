package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/config"
)

// TestNewCache tests the NewCache function
func TestNewCache(t *testing.T) {
	// Cannot use t.Parallel(): tests share global Instance via NewCache

	tests := []struct {
		name    string
		config  *config.Type
		wantErr bool
	}{
		{
			name:    "default_config",
			config:  &config.Type{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global Instance before each test
			Instance = nil

			cache, err := NewCache(tt.config)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, cache)
			require.NotNil(t, cache.i)
			// Verify global Instance was set
			require.Equal(t, cache, Instance)
		})
	}
}

// TestCacheSet tests the Set method
func TestCacheSet(t *testing.T) {
	// Cannot use t.Parallel(): tests share global Instance via NewCache
	cache, err := NewCache(&config.Type{})
	require.NoError(t, err)
	require.NotNil(t, cache)

	tests := []struct {
		name  string
		key   string
		value any
		cost  int64
	}{
		{
			name:  "string_value",
			key:   "string_key",
			value: "hello",
			cost:  1,
		},
		{
			name:  "int_value",
			key:   "int_key",
			value: 42,
			cost:  1,
		},
		{
			name:  "struct_value",
			key:   "struct_key",
			value: struct{ Name string }{Name: "test"},
			cost:  1,
		},
		{
			name:  "nil_value",
			key:   "nil_key",
			value: nil,
			cost:  1,
		},
		{
			name:  "zero_cost",
			key:   "zero_cost_key",
			value: "value",
			cost:  0,
		},
		{
			name:  "large_cost",
			key:   "large_cost_key",
			value: "value",
			cost:  1 << 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cache.SetRaw(tt.key, tt.value, tt.cost)
			require.True(t, result, "Set should return true")
			cache.Wait()
		})
	}
}

// TestCacheSetWithTTL tests the SetWithTTL method
func TestCacheSetWithTTL(t *testing.T) {
	// Cannot use t.Parallel(): tests share global Instance via NewCache
	cache, err := NewCache(&config.Type{})
	require.NoError(t, err)
	require.NotNil(t, cache)

	tests := []struct {
		name  string
		key   string
		value any
		cost  int64
		ttl   time.Duration
	}{
		{
			name:  "short_ttl",
			key:   "short_ttl_key",
			value: "expires soon",
			cost:  1,
			ttl:   10 * time.Millisecond,
		},
		{
			name:  "long_ttl",
			key:   "long_ttl_key",
			value: "expires later",
			cost:  1,
			ttl:   time.Hour,
		},
		{
			name:  "one_second_ttl",
			key:   "one_second_key",
			value: 123,
			cost:  1,
			ttl:   time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cache.SetWithTTL(tt.key, tt.value, tt.cost, tt.ttl)
			require.True(t, result, "SetWithTTL should return true")
			cache.Wait()
		})
	}
}

// TestCacheGet tests the Get method
func TestCacheGet(t *testing.T) {
	// Cannot use t.Parallel(): tests share global Instance via NewCache
	cache, err := NewCache(&config.Type{})
	require.NoError(t, err)
	require.NotNil(t, cache)

	// Set up test data
	stringValue := "test_string"
	intValue := 42
	structValue := struct{ Name string }{Name: "test"}

	cache.SetRaw("string_key", stringValue, 1)
	cache.SetRaw("int_key", intValue, 1)
	cache.SetRaw("struct_key", structValue, 1)
	cache.Wait()

	tests := []struct {
		name      string
		key       string
		wantValue any
		wantOK    bool
	}{
		{
			name:      "existing_string",
			key:       "string_key",
			wantValue: stringValue,
			wantOK:    true,
		},
		{
			name:      "existing_int",
			key:       "int_key",
			wantValue: intValue,
			wantOK:    true,
		},
		{
			name:      "existing_struct",
			key:       "struct_key",
			wantValue: structValue,
			wantOK:    true,
		},
		{
			name:      "missing_key",
			key:       "nonexistent_key",
			wantValue: nil,
			wantOK:    false,
		},
		{
			name:      "empty_key",
			key:       "",
			wantValue: nil,
			wantOK:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValue, gotOK := cache.GetRaw(tt.key)
			require.Equal(t, tt.wantOK, gotOK, "Get ok mismatch")
			if tt.wantOK {
				require.Equal(t, tt.wantValue, gotValue)
			}
		})
	}
}

// TestCacheDel tests the Del method
func TestCacheDel(t *testing.T) {
	// Cannot use t.Parallel(): tests share global Instance via NewCache
	cache, err := NewCache(&config.Type{})
	require.NoError(t, err)
	require.NotNil(t, cache)

	// Set up test data
	cache.SetRaw("key1", "value1", 1)
	cache.SetRaw("key2", "value2", 1)
	cache.Wait()

	tests := []struct {
		name       string
		delKey     string
		checkKey   string
		checkValue any
		checkOK    bool
	}{
		{
			name:       "delete_existing_key",
			delKey:     "key1",
			checkKey:   "key1",
			checkValue: nil,
			checkOK:    false,
		},
		{
			name:       "delete_again",
			delKey:     "key1",
			checkKey:   "key1",
			checkValue: nil,
			checkOK:    false,
		},
		{
			name:       "delete_missing_key",
			delKey:     "nonexistent",
			checkKey:   "nonexistent",
			checkValue: nil,
			checkOK:    false,
		},
		{
			name:       "key2_still_exists",
			delKey:     "key1",
			checkKey:   "key2",
			checkValue: "value2",
			checkOK:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache.DelRaw(tt.delKey)
			cache.Wait()
			gotValue, gotOK := cache.GetRaw(tt.checkKey)
			require.Equal(t, tt.checkOK, gotOK, "Get ok mismatch after Del")
			require.Equal(t, tt.checkValue, gotValue)
		})
	}
}

// TestCacheWait tests the Wait method
func TestCacheWait(t *testing.T) {
	// Cannot use t.Parallel(): tests share global Instance via NewCache

	t.Run("multiple values do not panic", func(t *testing.T) {
		cache, err := NewCache(&config.Type{})
		require.NoError(t, err)
		require.NotNil(t, cache)

		for i := range 100 {
			cache.SetRaw(string(rune('a'+i%26)), i, 1)
		}

		require.NotPanics(t, func() {
			cache.Wait()
		})
	})
}

// TestCacheIntegration tests basic cache operations together
func TestCacheIntegration(t *testing.T) {
	// Cannot use t.Parallel(): tests share global Instance via NewCache

	t.Run("set get del integration", func(t *testing.T) {
		cache, err := NewCache(&config.Type{})
		require.NoError(t, err)
		require.NotNil(t, cache)

		ok := cache.SetRaw("integration_key", "integration_value", 1)
		require.True(t, ok)
		cache.Wait()

		value, ok := cache.GetRaw("integration_key")
		require.True(t, ok)
		require.Equal(t, "integration_value", value)

		cache.DelRaw("integration_key")
		cache.Wait()

		value, ok = cache.GetRaw("integration_key")
		require.False(t, ok)
		require.Nil(t, value)

		cache.Wait()
	})
}

// TestCacheStringCache tests the StringCache interface implementation on the Ristretto-backed Cache.
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
		require.Empty(t, val)
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

func TestCacheDelByPrefix(t *testing.T) {
	cache, err := NewCache(&config.Type{})
	require.NoError(t, err)
	require.NotNil(t, cache)
	defer cache.Wait()

	tests := []struct {
		name     string
		setup    func(c *Cache)
		prefix   string
		wantKeys map[string]bool
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

func TestCacheTTLExpiration(t *testing.T) {
	// Cannot use t.Parallel(): tests share global Instance via NewCache

	tests := []struct {
		name      string
		ttl       time.Duration
		wait      time.Duration
		wantExist bool
	}{
		{
			name:      "short TTL expires",
			ttl:       20 * time.Millisecond,
			wait:      50 * time.Millisecond,
			wantExist: false,
		},
		{
			name:      "long TTL still valid",
			ttl:       time.Second,
			wait:      10 * time.Millisecond,
			wantExist: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache, err := NewCache(&config.Type{})
			require.NoError(t, err)
			require.NotNil(t, cache)

			cache.SetWithTTL("ttl_key", "ttl_value", 1, tt.ttl)
			cache.Wait()

			time.Sleep(tt.wait)

			value, ok := cache.GetRaw("ttl_key")
			if tt.wantExist {
				require.True(t, ok)
				require.Equal(t, "ttl_value", value)
			} else {
				require.False(t, ok)
				require.Nil(t, value)
			}
		})
	}
}
