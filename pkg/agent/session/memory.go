package session

import (
	"context"
	"fmt"
	"sync"
)

// MemoryStorage is an in-memory Storage implementation for tests and ephemeral runs.
type MemoryStorage struct {
	mu      sync.RWMutex
	entries []TreeEntry
	leafID  string
}

// NewMemoryStorage creates an empty in-memory session store.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{}
}

// Append stores a tree entry in insertion order.
func (m *MemoryStorage) Append(_ context.Context, entry TreeEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = append(m.entries, entry)
	return nil
}

// GetBranch returns the ordered path to the requested leaf.
func (m *MemoryStorage) GetBranch(_ context.Context, leafID string) ([]TreeEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if leafID == "" {
		return nil, fmt.Errorf("session memory: empty leaf id")
	}

	byID := make(map[string]TreeEntry, len(m.entries))
	for _, entry := range m.entries {
		byID[entry.ID] = entry
	}
	leaf, ok := byID[leafID]
	if !ok {
		return nil, fmt.Errorf("session memory: leaf %q not found", leafID)
	}

	path := []TreeEntry{leaf}
	current := leaf
	for current.ParentID != "" {
		parent, exists := byID[current.ParentID]
		if !exists {
			break
		}
		path = append([]TreeEntry{parent}, path...)
		current = parent
	}
	return path, nil
}

// GetLeafID returns the current leaf pointer.
func (m *MemoryStorage) GetLeafID(_ context.Context) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.leafID, nil
}

// SetLeafID updates the current leaf pointer.
func (m *MemoryStorage) SetLeafID(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.leafID = id
	return nil
}

// ListEntries returns all stored entries in append order.
func (m *MemoryStorage) ListEntries(_ context.Context) ([]TreeEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]TreeEntry(nil), m.entries...), nil
}

// Entries returns a snapshot of stored entries for assertions.
func (m *MemoryStorage) Entries() []TreeEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]TreeEntry(nil), m.entries...)
}
