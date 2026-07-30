// Package store provides database storage implementations.
package store

import (
	"context"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/clip"
)

// ---------------------------------------------------------------------------
// ClipStore
// ---------------------------------------------------------------------------

// ClipStore persists shareable markdown clips keyed by short slugs.
type ClipStore struct {
	client *gen.Client
}

// NewClipStore creates a ClipStore with the given ent client.
func NewClipStore(client *gen.Client) *ClipStore {
	return &ClipStore{client: client}
}

// ClipStoreFromDB returns a ClipStore using the global database client.
func ClipStoreFromDB() *ClipStore {
	return NewClipStore(ClientFromDB())
}

// CreateClip inserts a new clip row.
func (s *ClipStore) CreateClip(ctx context.Context, slug, title, description, content, createdBy string) error {
	if s == nil || s.client == nil {
		return nil
	}
	_, err := s.client.Clip.Create().
		SetSlug(slug).
		SetTitle(title).
		SetDescription(description).
		SetContent(content).
		SetCreatedBy(createdBy).
		Save(ctx)
	return err
}

// GetClipBySlug retrieves a clip by slug. Returns nil if not found.
func (s *ClipStore) GetClipBySlug(ctx context.Context, slug string) (*gen.Clip, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	row, err := s.client.Clip.Query().
		Where(clip.SlugEQ(slug)).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// ListClips returns clips ordered by created_at descending.
// When limit <= 0, all clips are returned.
func (s *ClipStore) ListClips(ctx context.Context, limit int) ([]*gen.Clip, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}
	q := s.client.Clip.Query().Order(gen.Desc(clip.FieldCreatedAt))
	if limit > 0 {
		q = q.Limit(limit)
	}
	return q.All(ctx)
}
