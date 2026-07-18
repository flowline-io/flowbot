package trilium

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/capability"
)

// ListQuery wraps pagination for listing notes.
type ListQuery = capability.NoteListQuery

// Service defines the note capability contract.
type Service interface {
	List(ctx context.Context, q *ListQuery) (*capability.ListResult[capability.Note], error)
	Get(ctx context.Context, id string) (*capability.Note, error)
	Create(ctx context.Context, title, content, typ, parentNoteID string) (*capability.Note, error)
	Update(ctx context.Context, id, title, content string) (*capability.Note, error)
	Delete(ctx context.Context, id string) error
	GetContent(ctx context.Context, id string) (string, error)
	SetContent(ctx context.Context, id, content string) error
	Search(ctx context.Context, query string) (*capability.ListResult[capability.Note], error)
	GetAppInfo(ctx context.Context) (*capability.Note, error)
	ListRawEvents(ctx context.Context, cursor string) ([]any, string, error)
	HealthCheck(ctx context.Context) (bool, error)
}
