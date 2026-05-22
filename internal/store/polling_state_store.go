// Package store provides database storage implementations.
package store

import (
	"context"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/pollingstate"
)

// PollingStateStore persists polling state entries for the provider event source framework.
type PollingStateStore struct {
	client *gen.Client
}

// NewPollingStateStore returns a PollingStateStore backed by the given Ent client.
func NewPollingStateStore(client *gen.Client) *PollingStateStore {
	return &PollingStateStore{client: client}
}

// PollingStateEntry represents a single persisted polling state row.
type PollingStateEntry struct {
	Cursor      string
	KnownHashes map[string]string
	UpdatedAt   any
}

// LoadAll loads all polling state entries from the database.
func (s *PollingStateStore) LoadAll(ctx context.Context) (map[string]PollingStateEntry, error) {
	if s == nil || s.client == nil {
		return make(map[string]PollingStateEntry), nil
	}
	rows, err := s.client.PollingState.Query().All(ctx)
	if err != nil {
		return nil, err
	}
	result := make(map[string]PollingStateEntry, len(rows))
	for _, row := range rows {
		result[row.ResourceName] = PollingStateEntry{
			Cursor:      row.Cursor,
			KnownHashes: row.KnownHashes,
			UpdatedAt:   row.UpdatedAt,
		}
	}
	return result, nil
}

// Save upserts a polling state entry for the given resource.
// If an entry with the same resource name already exists, it is updated; otherwise a new one is created.
func (s *PollingStateStore) Save(ctx context.Context, resourceName, cursor string, knownHashes map[string]string) error {
	if s == nil || s.client == nil {
		return nil
	}
	if knownHashes == nil {
		knownHashes = make(map[string]string)
	}
	existing, err := s.client.PollingState.Query().
		Where(pollingstate.ResourceName(resourceName)).
		Only(ctx)
	if err != nil && !gen.IsNotFound(err) {
		return err
	}
	if existing != nil {
		_, err = s.client.PollingState.UpdateOne(existing).
			SetCursor(cursor).
			SetKnownHashes(knownHashes).
			Save(ctx)
		return err
	}
	_, err = s.client.PollingState.Create().
		SetResourceName(resourceName).
		SetCursor(cursor).
		SetKnownHashes(knownHashes).
		Save(ctx)
	return err
}
