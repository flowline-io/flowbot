package cache

import (
	"context"
	"time"

	"github.com/dgraph-io/ristretto/v2"

	"github.com/flowline-io/flowbot/pkg/config"
)

var Instance *Cache

type Cache struct {
	i *ristretto.Cache[string, any]
}

func NewCache(_ *config.Type) (*Cache, error) {
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

func (c *Cache) SetRaw(key string, value any, cost int64) bool {
	return c.i.Set(key, value, cost)
}

func (c *Cache) SetWithTTL(key string, value any, cost int64, ttl time.Duration) bool {
	return c.i.SetWithTTL(key, value, cost, ttl)
}

func (c *Cache) GetRaw(key string) (any, bool) {
	return c.i.Get(key)
}

func (c *Cache) DelRaw(key string) {
	c.i.Del(key)
}

func (c *Cache) Wait() {
	c.i.Wait()
}

// Get retrieves a string value from the cache. Returns false if the key is not found.
func (c *Cache) Get(_ context.Context, key Key) (string, bool, error) {
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

// Set stores a string value with the given TTL.
func (c *Cache) Set(_ context.Context, key Key, value string, ttl TTL) error {
	c.i.SetWithTTL(key.String(), value, 1, ttl.Duration())
	return nil
}

// SetNX stores a value only if the key does not already exist. Returns true if the value was set.
func (c *Cache) SetNX(_ context.Context, key Key, value string, ttl TTL) (bool, error) {
	_, exists := c.i.Get(key.String())
	if exists {
		return false, nil
	}
	c.i.SetWithTTL(key.String(), value, 1, ttl.Duration())
	return true, nil
}

// Del removes a key from the cache.
func (c *Cache) Del(_ context.Context, key Key) error {
	c.i.Del(key.String())
	recordEviction("ristretto")
	return nil
}

// Exists checks whether a key is present in the cache.
func (c *Cache) Exists(_ context.Context, key Key) (bool, error) {
	_, ok := c.i.Get(key.String())
	return ok, nil
}

// Expire refreshes the TTL on an existing key.
func (c *Cache) Expire(_ context.Context, key Key, ttl TTL) error {
	val, ok := c.i.Get(key.String())
	if !ok {
		return nil
	}
	c.i.SetWithTTL(key.String(), val, 1, ttl.Duration())
	return nil
}
