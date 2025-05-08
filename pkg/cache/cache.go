package cache

import (
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"github.com/flowline-io/flowbot/pkg/config"
)

var Instance *Cache

type Cache struct {
	i *ristretto.Cache[string, any]
}

func NewCache(_ config.Type) (*Cache, error) {
	cache, err := ristretto.NewCache(&ristretto.Config[string, any]{
		NumCounters: 1e7,     // number of keys to track frequency of (10M).
		MaxCost:     1 << 30, // maximum cost of cache (1GB).
		BufferItems: 64,      // number of keys per Get buffer.
	})
	if err != nil {
		return nil, err
	}

	Instance = &Cache{i: cache}

	return Instance, nil
}

func (c *Cache) Set(key string, value any, cost int64) bool {
	return c.i.Set(key, value, cost)
}

func (c *Cache) SetWithTTL(key string, value any, cost int64, ttl time.Duration) bool {
	return c.i.SetWithTTL(key, value, cost, ttl)
}

func (c *Cache) Get(key string) (any, bool) {
	return c.i.Get(key)
}

func (c *Cache) Del(key string) {
	c.i.Del(key)
}

func (c *Cache) Wait() {
	c.i.Wait()
}
