package cache

import "fmt"

// Key represents a namespaced cache key following the format {prefix}:{entity}:{identifier}.
// It enforces consistent key naming across all cache operations.
type Key struct {
	// Prefix identifies the business domain (e.g. "online", "chat", "notify").
	Prefix string
	// Entity identifies the data purpose within the domain (e.g. "agent", "session", "throttle").
	Entity string
	// Identifier is the business primary key, which may contain additional colon-delimited segments.
	Identifier string
}

// NewKey creates a Key from its components.
func NewKey(prefix, entity, identifier string) Key {
	return Key{Prefix: prefix, Entity: entity, Identifier: identifier}
}

// String returns the colon-delimited key string used for cache operations.
func (k Key) String() string {
	return fmt.Sprintf("%s:%s:%s", k.Prefix, k.Entity, k.Identifier)
}
