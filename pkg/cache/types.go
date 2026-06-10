package cache

import "context"

// StringCache covers raw string KV operations, backed by either Ristretto or Redis.
// It is the foundational interface that every backend must satisfy.
type StringCache interface {
	// Get retrieves the string value for key. Returns false if not present.
	Get(ctx context.Context, key Key) (string, bool, error)
	// Set stores a string value with the given TTL.
	Set(ctx context.Context, key Key, value string, ttl TTL) error
	// SetNX sets the value only if the key does not already exist.
	// Returns true if the value was set.
	// Implementations should provide atomic check-and-set semantics.
	SetNX(ctx context.Context, key Key, value string, ttl TTL) (bool, error)
	// Del removes the key from the cache.
	Del(ctx context.Context, key Key) error
	// Exists returns whether the key is present in the cache.
	Exists(ctx context.Context, key Key) (bool, error)
	// Expire sets or updates the TTL on an existing key.
	Expire(ctx context.Context, key Key, ttl TTL) error
}

// IntCache covers integer counters, backed by Redis.
// It provides atomic increment operations suitable for rate limiting, statistics, and gauges.
type IntCache interface {
	// GetInt64 retrieves the integer value for key.
	GetInt64(ctx context.Context, key Key) (int64, error)
	// SetInt64 stores an integer value with the given TTL.
	SetInt64(ctx context.Context, key Key, value int64, ttl TTL) error
	// Incr atomically increments the key and returns the new value.
	Incr(ctx context.Context, key Key) (int64, error)
	// IncrWithTTL atomically increments the key, sets the TTL if the key is new,
	// and returns the new value.
	IncrWithTTL(ctx context.Context, key Key, ttl TTL) (int64, error)
}

// SetCache covers set-based deduplication, backed by Redis.
// It manages unordered collections of unique members with O(1) membership tests.
type SetCache interface {
	// Add adds one or more members to the set. Returns the number of members added.
	Add(ctx context.Context, key Key, ttl TTL, members ...string) (int64, error)
	// IsMember reports whether member is in the set.
	IsMember(ctx context.Context, key Key, member string) (bool, error)
	// Members returns all members of the set.
	Members(ctx context.Context, key Key) ([]string, error)
	// Remove removes one or more members from the set. Returns the number removed.
	Remove(ctx context.Context, key Key, members ...string) (int64, error)
	// Clear removes all members from the set.
	Clear(ctx context.Context, key Key) error
}

// ListCache covers list-based aggregation buffers, backed by Redis.
// It provides ordered insertion and ranged retrieval suitable for message queues
// and event buffers.
type ListCache interface {
	// Push appends values to the tail of the list. Returns the length after push.
	Push(ctx context.Context, key Key, values ...string) (int64, error)
	// Range returns elements from start to stop (inclusive, zero-indexed).
	// Use -1 for stop to read to the end.
	Range(ctx context.Context, key Key, start, stop int64) ([]string, error)
	// Len returns the current length of the list.
	Len(ctx context.Context, key Key) (int64, error)
	// Clear removes all elements from the list.
	Clear(ctx context.Context, key Key) error
}
