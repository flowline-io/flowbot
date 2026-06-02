package types

import "sync"

// FilterCache holds in-memory unique sets of sources and event types
// used to populate filter dropdowns without querying the database.
type FilterCache struct {
	mu         sync.RWMutex
	sources    []string
	eventTypes []string
	sourceSet  map[string]struct{}
	typeSet    map[string]struct{}
}

// NewFilterCache creates an empty FilterCache.
func NewFilterCache() *FilterCache {
	return &FilterCache{
		sourceSet: make(map[string]struct{}),
		typeSet:   make(map[string]struct{}),
	}
}

// SetSource adds a source to the cache if not already present.
func (fc *FilterCache) SetSource(source string) {
	if source == "" {
		return
	}
	fc.mu.Lock()
	defer fc.mu.Unlock()
	if _, ok := fc.sourceSet[source]; ok {
		return
	}
	fc.sourceSet[source] = struct{}{}
	fc.sources = append(fc.sources, source)
}

// SetEventType adds an event type to the cache if not already present.
func (fc *FilterCache) SetEventType(eventType string) {
	if eventType == "" {
		return
	}
	fc.mu.Lock()
	defer fc.mu.Unlock()
	if _, ok := fc.typeSet[eventType]; ok {
		return
	}
	fc.typeSet[eventType] = struct{}{}
	fc.eventTypes = append(fc.eventTypes, eventType)
}

// Sources returns a copy of all cached sources.
func (fc *FilterCache) Sources() []string {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	srcs := make([]string, len(fc.sources))
	copy(srcs, fc.sources)
	return srcs
}

// EventTypes returns a copy of all cached event types.
func (fc *FilterCache) EventTypes() []string {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	types := make([]string, len(fc.eventTypes))
	copy(types, fc.eventTypes)
	return types
}

// Hydrate populates the cache from database lists (deduplicates with existing).
func (fc *FilterCache) Hydrate(sources, eventTypes []string) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	for _, s := range sources {
		if _, ok := fc.sourceSet[s]; !ok {
			fc.sourceSet[s] = struct{}{}
			fc.sources = append(fc.sources, s)
		}
	}
	for _, t := range eventTypes {
		if _, ok := fc.typeSet[t]; !ok {
			fc.typeSet[t] = struct{}{}
			fc.eventTypes = append(fc.eventTypes, t)
		}
	}
}

// EventFilterCache is the global filter cache for event sources and types.
// Initialized by the web module on startup, updated by the store on event write.
var EventFilterCache = NewFilterCache()
