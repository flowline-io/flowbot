package syncx_test

import (
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"

	"github.com/flowline-io/flowbot/pkg/utils/syncx"
)

func TestGetNonExistent(t *testing.T) {
	m := syncx.Map[string, int]{}
	v, ok := m.Get("nothing")
	assert.False(t, ok)
	assert.Equal(t, 0, v)
}

func TestSetAndGet(t *testing.T) {
	m := syncx.Map[string, int]{}
	m.Set("somekey", 100)
	v, ok := m.Get("somekey")
	assert.True(t, ok)
	assert.Equal(t, 100, v)
}

func TestSetAndDelete(t *testing.T) {
	m := syncx.Map[string, int]{}
	m.Set("somekey", 100)
	v, ok := m.Get("somekey")
	assert.True(t, ok)
	assert.Equal(t, 100, v)
	m.Delete("somekey")
	v, ok = m.Get("somekey")
	assert.False(t, ok)
	assert.Equal(t, 0, v)
}

func TestConcurrentSetAndGet(t *testing.T) {
	m := syncx.Map[string, int]{}
	wg := sync.WaitGroup{}
	wg.Add(1000)
	for i := 1; i <= 1000; i++ {
		go func(ix int) {
			defer wg.Done()
			// introduce some arbitrary latency
			time.Sleep(time.Millisecond * time.Duration(rand.Intn(100)+1))
			m.Set("somekey", ix)
			v, ok := m.Get("somekey")
			assert.True(t, ok)
			assert.Positive(t, v)
		}(i)
	}
	wg.Wait()
}

func TestIterate(t *testing.T) {
	m := syncx.Map[string, int]{}
	m.Set("k1", 100)
	m.Set("k2", 200)
	vals := make([]int, 0)
	keys := make([]string, 0)
	m.Iterate(func(k string, v int) {
		vals = append(vals, v)
		keys = append(keys, k)
	})
	slices.Sort(vals)
	slices.Sort(keys)
	assert.Equal(t, []int{100, 200}, vals)
	assert.Equal(t, []string{"k1", "k2"}, keys)
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

func FuzzMapSetGet(f *testing.F) {
	f.Add("foo", 0)
	f.Add("bar", 42)
	f.Add("", -1)

	f.Fuzz(func(t *testing.T, key string, value int) {
		m := syncx.Map[string, int]{}
		m.Set(key, value)

		v, ok := m.Get(key)
		assert.True(t, ok, "Get after Set should return ok")
		assert.Equal(t, value, v, "Get after Set should return the same value")

		m.Delete(key)
		v, ok = m.Get(key)
		assert.False(t, ok, "Get after Delete should return !ok")
		assert.Equal(t, 0, v, "Get after Delete should return zero value")
	})
}

func FuzzMapIterate(f *testing.F) {
	f.Add([]byte(`["a"]`), []byte(`[1]`))
	f.Add([]byte(`["a","b","c"]`), []byte(`[1,2,3]`))
	f.Add([]byte(`[]`), []byte(`[]`))

	f.Fuzz(func(t *testing.T, keysData, valuesData []byte) {
		var keys []string
		var values []int
		if err := sonic.Unmarshal(keysData, &keys); err != nil {
			t.Skip()
		}
		if err := sonic.Unmarshal(valuesData, &values); err != nil {
			t.Skip()
		}
		if len(keys) != len(values) {
			t.Skip()
		}

		m := syncx.Map[string, int]{}
		dedup := make(map[string]int, len(keys))
		for i, k := range keys {
			m.Set(k, values[i])
			dedup[k] = values[i]
		}

		seen := make(map[string]int, len(keys))
		m.Iterate(func(k string, v int) {
			seen[k] = v
		})

		for k, expectedV := range dedup {
			v, ok := seen[k]
			assert.True(t, ok, "Iterate missed key %q", k)
			assert.Equal(t, expectedV, v, "Iterate wrong value for key %q", k)
		}
		assert.Len(t, seen, len(dedup), "Iterate saw %d entries, expected %d", len(seen), len(dedup))
	})
}
