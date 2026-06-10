package session

import "context"

// Storage persists session tree entries without prescribing a backend.
type Storage interface {
	Append(ctx context.Context, entry TreeEntry) error
	GetBranch(ctx context.Context, leafID string) ([]TreeEntry, error)
	GetLeafID(ctx context.Context) (string, error)
	SetLeafID(ctx context.Context, id string) error
}
