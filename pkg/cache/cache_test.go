package cache

import (
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/stretchr/testify/require"
)

// TestNewCache tests the NewCache function
func TestNewCache(t *testing.T) {
	tests := []struct {
		name    string
		config  config.Type
		wantErr bool
	}{
		{
			name:    "default_config",
			config:  config.Type{},
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
	cache, err := NewCache(config.Type{})
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
			result := cache.Set(tt.key, tt.value, tt.cost)
			require.True(t, result, "Set should return true")
			cache.Wait()
		})
	}
}

// TestCacheSetWithTTL tests the SetWithTTL method
func TestCacheSetWithTTL(t *testing.T) {
	cache, err := NewCache(config.Type{})
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
	cache, err := NewCache(config.Type{})
	require.NoError(t, err)
	require.NotNil(t, cache)

	// Set up test data
	stringValue := "test_string"
	intValue := 42
	structValue := struct{ Name string }{Name: "test"}

	cache.Set("string_key", stringValue, 1)
	cache.Set("int_key", intValue, 1)
	cache.Set("struct_key", structValue, 1)
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
			gotValue, gotOK := cache.Get(tt.key)
			require.Equal(t, tt.wantOK, gotOK, "Get ok mismatch")
			if tt.wantOK {
				require.Equal(t, tt.wantValue, gotValue)
			}
		})
	}
}

// TestCacheDel tests the Del method
func TestCacheDel(t *testing.T) {
	cache, err := NewCache(config.Type{})
	require.NoError(t, err)
	require.NotNil(t, cache)

	// Set up test data
	cache.Set("key1", "value1", 1)
	cache.Set("key2", "value2", 1)
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
			cache.Del(tt.delKey)
			cache.Wait()
			gotValue, gotOK := cache.Get(tt.checkKey)
			require.Equal(t, tt.checkOK, gotOK, "Get ok mismatch after Del")
			require.Equal(t, tt.checkValue, gotValue)
		})
	}
}

// TestCacheWait tests the Wait method
func TestCacheWait(t *testing.T) {
	cache, err := NewCache(config.Type{})
	require.NoError(t, err)
	require.NotNil(t, cache)

	// Set multiple values
	for i := 0; i < 100; i++ {
		cache.Set(string(rune('a'+i%26)), i, 1)
	}

	// Wait for all operations to complete
	require.NotPanics(t, func() {
		cache.Wait()
	})
}

// TestCacheIntegration tests basic cache operations together
func TestCacheIntegration(t *testing.T) {
	cache, err := NewCache(config.Type{})
	require.NoError(t, err)
	require.NotNil(t, cache)

	// Set a value
	ok := cache.Set("integration_key", "integration_value", 1)
	require.True(t, ok)
	cache.Wait()

	// Get the value
	value, ok := cache.Get("integration_key")
	require.True(t, ok)
	require.Equal(t, "integration_value", value)

	// Delete the value
	cache.Del("integration_key")
	cache.Wait()

	// Verify deletion
	value, ok = cache.Get("integration_key")
	require.False(t, ok)
	require.Nil(t, value)

	// Wait for all operations to complete
	cache.Wait()
}

// TestCacheTTLExpiration tests that TTL actually expires
func TestCacheTTLExpiration(t *testing.T) {
	cache, err := NewCache(config.Type{})
	require.NoError(t, err)
	require.NotNil(t, cache)

	// Set with very short TTL
	cache.SetWithTTL("ttl_key", "ttl_value", 1, 20*time.Millisecond)
	cache.Wait()

	// Should be present immediately
	value, ok := cache.Get("ttl_key")
	require.True(t, ok)
	require.Equal(t, "ttl_value", value)

	// Wait for TTL to expire
	time.Sleep(50 * time.Millisecond)

	// Should be gone after TTL
	value, ok = cache.Get("ttl_key")
	require.False(t, ok)
	require.Nil(t, value)
}
