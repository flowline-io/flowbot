// Package syncx provides synchronized concurrent-safe data structures.
package syncx

import "sync"

type Map[K comparable, V any] struct {
	m sync.Map
}

func (m *Map[K, V]) Delete(key K) {
	m.m.Delete(key)
}

func (m *Map[K, V]) Get(key K) (value V, ok bool) {
	v, loaded := m.m.Load(key)
	if !loaded {
		return value, ok
	}
	value, ok = v.(V)
	return
}

func (m *Map[K, V]) Set(key K, value V) {
	m.m.Store(key, value)
}

// LoadOrStore returns the existing value for the key if present.
// Otherwise, it stores and returns the given value.
// The loaded result is true if the value was loaded, false if stored.
func (m *Map[K, V]) LoadOrStore(key K, value V) (actual V, loaded bool) {
	v, ok := m.m.LoadOrStore(key, value)
	if ok {
		actual, loaded = v.(V)
		return actual, true
	}
	return value, false
}

// LoadAndDelete atomically retrieves and removes the value for the given key.
// It returns the value (if loaded) and whether it was loaded.
func (m *Map[K, V]) LoadAndDelete(key K) (value V, loaded bool) {
	v, ok := m.m.LoadAndDelete(key)
	if !ok {
		return value, false
	}
	value, ok = v.(V)
	if !ok {
		var zero V
		return zero, false
	}
	return value, true
}

func (m *Map[K, V]) Iterate(f func(key K, value V)) {
	m.m.Range(func(key, value any) bool {
		k, ok := key.(K)
		if !ok {
			return true
		}
		v, ok := value.(V)
		if !ok {
			return true
		}
		f(k, v)
		return true
	})
}
