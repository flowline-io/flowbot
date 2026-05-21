package syncx_test

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"

	"github.com/flowline-io/flowbot/pkg/utils/syncx"
)

func TestGetNonExistent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		setup  func(m *syncx.Map[string, int])
		key    string
		wantV  int
		wantOK bool
	}{
		{
			name:   "happy_path_get_from_empty_map",
			setup:  func(_ *syncx.Map[string, int]) {},
			key:    "nothing",
			wantV:  0,
			wantOK: false,
		},
		{
			name: "edge_get_after_delete",
			setup: func(m *syncx.Map[string, int]) {
				m.Set("temp", 42)
				m.Delete("temp")
			},
			key:    "temp",
			wantV:  0,
			wantOK: false,
		},
		{
			name: "edge_get_wrong_key_from_populated_map",
			setup: func(m *syncx.Map[string, int]) {
				m.Set("present", 99)
			},
			key:    "missing",
			wantV:  0,
			wantOK: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := syncx.Map[string, int]{}
			tt.setup(&m)
			v, ok := m.Get(tt.key)
			assert.False(t, ok)
			assert.Equal(t, tt.wantV, v)
		})
	}
}

func TestSetAndGet(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		key   string
		value int
		extra func(m *syncx.Map[string, int])
	}{
		{
			name:  "happy_path_set_and_get",
			key:   "somekey",
			value: 100,
		},
		{
			name:  "edge_overwrite_existing_key",
			key:   "somekey",
			value: 200,
			extra: func(m *syncx.Map[string, int]) {
				m.Set("somekey", 100)
			},
		},
		{
			name:  "edge_zero_value",
			key:   "zerokey",
			value: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := syncx.Map[string, int]{}
			if tt.extra != nil {
				tt.extra(&m)
			}
			m.Set(tt.key, tt.value)
			v, ok := m.Get(tt.key)
			assert.True(t, ok)
			assert.Equal(t, tt.value, v)
		})
	}
}

func TestSetAndDelete(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		key        string
		value      int
		deleteKey  string
		preSet     bool
		wantExists bool
		wantVal    int
	}{
		{
			name:       "happy_path_set_get_delete_get",
			key:        "somekey",
			value:      100,
			deleteKey:  "somekey",
			preSet:     true,
			wantExists: false,
			wantVal:    0,
		},
		{
			name:       "edge_delete_nonexistent_key",
			key:        "somekey",
			value:      100,
			deleteKey:  "otherkey",
			preSet:     true,
			wantExists: true,
			wantVal:    100,
		},
		{
			name:       "edge_delete_then_reinsert",
			key:        "somekey",
			value:      200,
			deleteKey:  "somekey",
			preSet:     true,
			wantExists: false,
			wantVal:    0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := syncx.Map[string, int]{}
			if tt.preSet {
				m.Set(tt.key, tt.value)
				v, ok := m.Get(tt.key)
				assert.True(t, ok)
				assert.Equal(t, tt.value, v)
			}
			m.Delete(tt.deleteKey)
			v, ok := m.Get(tt.key)
			assert.Equal(t, tt.wantExists, ok)
			assert.Equal(t, tt.wantVal, v)
		})
	}
}

func TestConcurrentSetAndGet(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		goroutines int
	}{
		{
			name:       "happy_path_high_concurrency",
			goroutines: 1000,
		},
		{
			name:       "edge_low_concurrency",
			goroutines: 10,
		},
		{
			name:       "edge_medium_concurrency",
			goroutines: 100,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := syncx.Map[string, int]{}
			wg := sync.WaitGroup{}
			wg.Add(tt.goroutines)
			for i := 1; i <= tt.goroutines; i++ {
				go func(ix int) {
					defer wg.Done()
					time.Sleep(time.Millisecond)
					m.Set("somekey", ix)
					v, ok := m.Get("somekey")
					assert.True(t, ok)
					assert.Positive(t, v)
				}(i)
			}
			wg.Wait()
		})
	}
}

func TestIterate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		entries  map[string]int
		wantKeys []string
		wantVals []int
	}{
		{
			name:     "happy_path_iterate_with_entries",
			entries:  map[string]int{"k1": 100, "k2": 200},
			wantKeys: []string{"k1", "k2"},
			wantVals: []int{100, 200},
		},
		{
			name:     "edge_iterate_empty_map",
			entries:  map[string]int{},
			wantKeys: []string{},
			wantVals: []int{},
		},
		{
			name:     "edge_iterate_single_entry",
			entries:  map[string]int{"only": 42},
			wantKeys: []string{"only"},
			wantVals: []int{42},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := syncx.Map[string, int]{}
			for k, v := range tt.entries {
				m.Set(k, v)
			}
			vals := make([]int, 0)
			keys := make([]string, 0)
			m.Iterate(func(k string, v int) {
				vals = append(vals, v)
				keys = append(keys, k)
			})
			slices.Sort(vals)
			slices.Sort(keys)
			assert.Equal(t, tt.wantVals, vals)
			assert.Equal(t, tt.wantKeys, keys)
		})
	}
}

func TestLoadAndDelete(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		setup  func(m *syncx.Map[string, int])
		key    string
		wantV  int
		wantOK bool
	}{
		{
			name:   "happy_path_load_and_delete_existing_entry",
			setup:  func(m *syncx.Map[string, int]) { m.Set("somekey", 42) },
			key:    "somekey",
			wantV:  42,
			wantOK: true,
		},
		{
			name:   "edge_load_and_delete_nonexistent_entry",
			setup:  func(_ *syncx.Map[string, int]) {},
			key:    "missing",
			wantV:  0,
			wantOK: false,
		},
		{
			name:   "edge_load_and_delete_twice_returns_nil_on_second",
			setup:  func(m *syncx.Map[string, int]) { m.Set("once", 99) },
			key:    "once",
			wantV:  0,
			wantOK: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := syncx.Map[string, int]{}
			tt.setup(&m)
			if tt.name == "edge_load_and_delete_twice_returns_nil_on_second" {
				v, ok := m.LoadAndDelete(tt.key)
				assert.True(t, ok)
				assert.Equal(t, 99, v)
				v, ok = m.LoadAndDelete(tt.key)
				assert.False(t, ok)
				assert.Equal(t, 0, v)
				return
			}
			v, ok := m.LoadAndDelete(tt.key)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantV, v)
			if tt.wantOK {
				_, ok := m.Get(tt.key)
				assert.False(t, ok)
			}
		})
	}
}

func BenchmarkSetAndGet(b *testing.B) {
	for i := 0; i < b.N; i++ {
		m := syncx.Map[string, int]{}
		m.Set("somekey", 100)
		v, ok := m.Get("somekey")
		assert.True(b, ok)
		assert.Equal(b, 100, v)
	}
}
