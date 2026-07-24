package server

import (
	"context"

	storepkg "github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/capability/core"
)

// initClipAbility wires clip persistence and other CapCore deps (registration is deferred to notify OnStart).
func initClipAbility() error {
	if storepkg.Database != nil {
		if client, ok := storepkg.Database.GetDB().(*storepkg.Client); ok && client != nil {
			core.SetPersister(&clipStorePersister{store: storepkg.NewClipStore(client)})
		}
		wireCoreKVStore()
	}
	wireCoreExecProvider()
	return nil
}

func registerCoreCapability() error {
	return core.Register()
}

// clipStorePersister adapts store.ClipStore to core.Persister.
type clipStorePersister struct {
	store *storepkg.ClipStore
}

// CreateClip inserts a clip row.
func (p *clipStorePersister) CreateClip(ctx context.Context, slug, title, description, content, createdBy string) error {
	return p.store.CreateClip(ctx, slug, title, description, content, createdBy)
}

// GetClipBySlug loads a clip by slug.
func (p *clipStorePersister) GetClipBySlug(ctx context.Context, slug string) (*core.Record, error) {
	row, err := p.store.GetClipBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	return &core.Record{
		Slug:        row.Slug,
		Title:       row.Title,
		Description: row.Description,
		Content:     row.Content,
		CreatedBy:   row.CreatedBy,
		CreatedAt:   row.CreatedAt,
	}, nil
}
