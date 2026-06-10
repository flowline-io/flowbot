package ability

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sync"
	"time"

	"github.com/flowline-io/flowbot/pkg/flog"
)

// PollingState manages in-memory polling state with periodic persistence.
// Each pollEntry has its own lock to avoid global contention.
type PollingState struct {
	mu      sync.RWMutex
	entries map[string]*pollingEntryState
	backend Persistence
	dirty   map[string]bool
}

type pollingEntryState struct {
	mu    sync.Mutex
	entry PollingEntry
}

// NewPollingState creates a PollingState backed by the given Persistence.
func NewPollingState(backend Persistence) *PollingState {
	return &PollingState{
		entries: make(map[string]*pollingEntryState),
		backend: backend,
		dirty:   make(map[string]bool),
	}
}

// Get returns a copy of the polling entry for the named resource.
// Returns an empty entry if the resource is unknown.
func (s *PollingState) Get(name string) PollingEntry {
	s.mu.RLock()
	e, ok := s.entries[name]
	s.mu.RUnlock()
	if !ok {
		return PollingEntry{KnownHashes: make(map[string]string)}
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return PollingEntry{
		Cursor:      e.entry.Cursor,
		KnownHashes: copyMap(e.entry.KnownHashes),
		UpdatedAt:   e.entry.UpdatedAt,
	}
}

// Update sets the polling entry for the named resource.
func (s *PollingState) Update(name string, entry PollingEntry) {
	s.mu.Lock()
	e, ok := s.entries[name]
	if !ok {
		e = &pollingEntryState{}
		s.entries[name] = e
	}
	s.mu.Unlock()

	e.mu.Lock()
	e.entry = PollingEntry{
		Cursor:      entry.Cursor,
		KnownHashes: copyMap(entry.KnownHashes),
		UpdatedAt:   time.Now(),
	}
	e.mu.Unlock()

	s.mu.Lock()
	s.dirty[name] = true
	s.mu.Unlock()
}

// MarkDirty marks a resource as needing persistence.
func (s *PollingState) MarkDirty(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dirty[name] = true
}

// Flush persists all dirty entries to the backend.
// It attempts to save every dirty entry and collects errors so that a single
// failure does not abandon remaining entries.
func (s *PollingState) Flush(ctx context.Context) error {
	s.mu.RLock()
	names := make([]string, 0, len(s.dirty))
	for name := range s.dirty {
		names = append(names, name)
	}
	s.mu.RUnlock()

	var errs []error
	for _, name := range names {
		s.mu.RLock()
		e, ok := s.entries[name]
		s.mu.RUnlock()
		if !ok {
			s.mu.Lock()
			delete(s.dirty, name)
			s.mu.Unlock()
			continue
		}
		e.mu.Lock()
		entry := PollingEntry{
			Cursor:      e.entry.Cursor,
			KnownHashes: copyMap(e.entry.KnownHashes),
		}
		e.mu.Unlock()

		if s.backend != nil {
			if err := s.backend.Save(ctx, name, entry.Cursor, entry.KnownHashes); err != nil {
				errs = append(errs, err)
				continue
			}
		}

		s.mu.Lock()
		delete(s.dirty, name)
		s.mu.Unlock()
	}

	if len(errs) > 0 {
		return fmt.Errorf("flush errors: %w", errors.Join(errs...))
	}
	return nil
}

// Load restores state from the persistence backend.
// It overwrites any in-memory entries with the persisted data — this is
// intended to be called once during startup before any polls run, so
// persisted state takes precedence over in-memory defaults.
func (s *PollingState) Load(ctx context.Context) error {
	if s.backend == nil {
		return nil
	}
	persisted, err := s.backend.LoadAll(ctx)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for name, pentry := range persisted {
		s.entries[name] = &pollingEntryState{
			entry: PollingEntry{
				Cursor:      pentry.Cursor,
				KnownHashes: copyMap(pentry.KnownHashes),
				UpdatedAt:   pentry.UpdatedAt,
			},
		}
	}
	return nil
}

// FlushInterval returns the recommended interval between periodic flushes.
func (*PollingState) FlushInterval() time.Duration {
	return 5 * time.Minute
}

// startFlushLoop runs a background goroutine that periodically flushes dirty state.
func (m *EventSourceManager) startFlushLoop(ctx context.Context) {
	if m.stateStore == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(m.stateStore.FlushInterval())
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				start := time.Now()
				if err := m.stateStore.Flush(context.Background()); err != nil {
					flog.Warn("event_source: periodic flush failed: %v", err)
				}
				if m.metrics != nil {
					m.metrics.ObserveStateFlushDuration(time.Since(start).Seconds())
				}
			}
		}
	}()
}

func copyMap(src map[string]string) map[string]string {
	if src == nil {
		return make(map[string]string)
	}
	dst := make(map[string]string, len(src))
	maps.Copy(dst, src)
	return dst
}
