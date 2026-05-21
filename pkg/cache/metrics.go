package cache

import "github.com/flowline-io/flowbot/pkg/stats"

// recordHit increments the cache hit counter for the given backend.
func recordHit(backend string) {
	stats.CacheHitTotalCounter(backend).Inc()
}

// recordMiss increments the cache miss counter for the given backend.
func recordMiss(backend string) {
	stats.CacheMissTotalCounter(backend).Inc()
}

// recordEviction increments the cache eviction counter for the given backend.
func recordEviction(backend string) {
	stats.CacheEvictionTotalCounter(backend).Inc()
}
