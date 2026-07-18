package server

import (
	"context"

	storepkg "github.com/flowline-io/flowbot/internal/store"
	abilityclip "github.com/flowline-io/flowbot/pkg/capability/clip"
)

// initClipAbility wires clip persistence and registers the clip capability.
func initClipAbility() error {
	if storepkg.Database != nil {
		if client, ok := storepkg.Database.GetDB().(*storepkg.Client); ok && client != nil {
			abilityclip.SetPersister(&clipStorePersister{store: storepkg.NewClipStore(client)})
		}
	}
	return abilityclip.Register()
}

// clipStorePersister adapts store.ClipStore to capability/clip.Persister.
type clipStorePersister struct {
	store *storepkg.ClipStore
}

// CreateClip inserts a clip row.
func (p *clipStorePersister) CreateClip(ctx context.Context, slug, title, description, content, createdBy string) error {
	return p.store.CreateClip(ctx, slug, title, description, content, createdBy)
}

// GetClipBySlug loads a clip by slug.
func (p *clipStorePersister) GetClipBySlug(ctx context.Context, slug string) (*abilityclip.Record, error) {
	row, err := p.store.GetClipBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	return &abilityclip.Record{
		Slug:        row.Slug,
		Title:       row.Title,
		Description: row.Description,
		Content:     row.Content,
		CreatedBy:   row.CreatedBy,
		CreatedAt:   row.CreatedAt,
	}, nil
}
