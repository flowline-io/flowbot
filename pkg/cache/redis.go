package cache

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// RedisStore wraps a Redis client to implement StringCache, IntCache, SetCache, and ListCache.
// All operations record cache hit/miss/eviction metrics via the helpers in metrics.go.
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore creates a RedisStore backed by a Redis client.
func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
}

// Get retrieves a string value by key. Returns false if the key is not found.
func (s *RedisStore) Get(ctx context.Context, key Key) (string, bool, error) {
	val, err := s.client.Get(ctx, key.String()).Result()
	if err == redis.Nil {
		recordMiss("redis")
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("redis get %s: %w", key.String(), err)
	}
	recordHit("redis")
	return val, true, nil
}

// Set stores a string value with the given TTL.
func (s *RedisStore) Set(ctx context.Context, key Key, value string, ttl TTL) error {
	return s.client.Set(ctx, key.String(), value, ttl.Duration()).Err()
}

// SetNX sets a key only if it does not already exist. Returns true if set, false if key already existed.
func (s *RedisStore) SetNX(ctx context.Context, key Key, value string, ttl TTL) (bool, error) {
	return s.client.SetNX(ctx, key.String(), value, ttl.Duration()).Result()
}

// Del removes a key from the cache.
func (s *RedisStore) Del(ctx context.Context, key Key) error {
	err := s.client.Del(ctx, key.String()).Err()
	if err != nil {
		return fmt.Errorf("redis del %s: %w", key.String(), err)
	}
	recordEviction("redis")
	return nil
}

// Exists checks whether a key is present.
func (s *RedisStore) Exists(ctx context.Context, key Key) (bool, error) {
	n, err := s.client.Exists(ctx, key.String()).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists %s: %w", key.String(), err)
	}
	if n > 0 {
		recordHit("redis")
		return true, nil
	}
	recordMiss("redis")
	return false, nil
}

// Expire refreshes the TTL on an existing key.
func (s *RedisStore) Expire(ctx context.Context, key Key, ttl TTL) error {
	return s.client.Expire(ctx, key.String(), ttl.Duration()).Err()
}

// GetInt64 retrieves an int64 value. Returns 0 if the key is not found.
func (s *RedisStore) GetInt64(ctx context.Context, key Key) (int64, error) {
	val, err := s.client.Get(ctx, key.String()).Int64()
	if err == redis.Nil {
		recordMiss("redis")
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("redis get int64 %s: %w", key.String(), err)
	}
	recordHit("redis")
	return val, nil
}

// SetInt64 stores an int64 value with the given TTL.
func (s *RedisStore) SetInt64(ctx context.Context, key Key, value int64, ttl TTL) error {
	return s.client.Set(ctx, key.String(), value, ttl.Duration()).Err()
}

// Incr atomically increments the integer at key by 1 and returns the new value.
func (s *RedisStore) Incr(ctx context.Context, key Key) (int64, error) {
	return s.client.Incr(ctx, key.String()).Result()
}

// IncrWithTTL atomically increments the integer at key by 1 and sets the TTL
// if the key was newly created (i.e., the new count is 1).
func (s *RedisStore) IncrWithTTL(ctx context.Context, key Key, ttl TTL) (int64, error) {
	newCount, err := s.client.Incr(ctx, key.String()).Result()
	if err != nil {
		return 0, fmt.Errorf("redis incr %s: %w", key.String(), err)
	}
	if newCount == 1 {
		if err := s.client.Expire(ctx, key.String(), ttl.Duration()).Err(); err != nil {
			return newCount, fmt.Errorf("redis expire %s: %w", key.String(), err)
		}
	}
	return newCount, nil
}

// Add adds members to a set. If the key is newly created, sets the TTL.
func (s *RedisStore) Add(ctx context.Context, key Key, ttl TTL, members ...string) (int64, error) {
	n, err := s.client.SAdd(ctx, key.String(), toAny(members)...).Result()
	if err != nil {
		return 0, fmt.Errorf("redis sadd %s: %w", key.String(), err)
	}
	if ttl.Duration() > 0 {
		s.client.Expire(ctx, key.String(), ttl.Duration())
	}
	return n, nil
}

// IsMember checks whether a member exists in the set.
func (s *RedisStore) IsMember(ctx context.Context, key Key, member string) (bool, error) {
	return s.client.SIsMember(ctx, key.String(), member).Result()
}

// Members returns all members of the set.
func (s *RedisStore) Members(ctx context.Context, key Key) ([]string, error) {
	return s.client.SMembers(ctx, key.String()).Result()
}

// Remove removes members from a set.
func (s *RedisStore) Remove(ctx context.Context, key Key, members ...string) (int64, error) {
	return s.client.SRem(ctx, key.String(), toAny(members)...).Result()
}

// Clear removes the entire set or list key.
func (s *RedisStore) Clear(ctx context.Context, key Key) error {
	return s.client.Del(ctx, key.String()).Err()
}

// Push appends values to the right end of a list. Returns the list length after the push.
func (s *RedisStore) Push(ctx context.Context, key Key, values ...string) (int64, error) {
	args := make([]any, len(values))
	for i, v := range values {
		args[i] = v
	}
	return s.client.RPush(ctx, key.String(), args...).Result()
}

// Range returns a range of elements from the list. Use 0, -1 for all elements.
func (s *RedisStore) Range(ctx context.Context, key Key, start, stop int64) ([]string, error) {
	return s.client.LRange(ctx, key.String(), start, stop).Result()
}

// Len returns the length of the list.
func (s *RedisStore) Len(ctx context.Context, key Key) (int64, error) {
	return s.client.LLen(ctx, key.String()).Result()
}

// ScanKeys scans for keys matching a pattern, returning all matching keys.
func (s *RedisStore) ScanKeys(ctx context.Context, pattern string, count int64) ([]string, error) {
	var keys []string
	var cursor uint64
	for {
		result, nextCursor, err := s.client.Scan(ctx, cursor, pattern, count).Result()
		if err != nil {
			return nil, fmt.Errorf("redis scan %s: %w", pattern, err)
		}
		keys = append(keys, result...)
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return keys, nil
}

// ExistsRaw checks whether a raw key string exists in Redis. Used for scanning
// and other operations where a pre-formatted key string is available.
func (s *RedisStore) ExistsRaw(ctx context.Context, key string) (bool, error) {
	n, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists %s: %w", key, err)
	}
	return n > 0, nil
}

// toAny converts a string slice to an any slice for go-redis variadic args.
func toAny(ss []string) []any {
	res := make([]any, len(ss))
	for i, s := range ss {
		res[i] = s
	}
	return res
}
