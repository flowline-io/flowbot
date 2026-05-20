package syncx // import "https://github.com/runabol/tork"

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
