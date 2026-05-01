package bookmark

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/ability"
)

type ListQuery struct {
	Page       ability.PageRequest
	Archived   *bool
	Favourited *bool
	Tags       []string
}

type SearchQuery struct {
	Page ability.PageRequest
	Q    string
}

type Service interface {
	List(ctx context.Context, q *ListQuery) (*ability.ListResult[ability.Bookmark], error)
	Get(ctx context.Context, id string) (*ability.Bookmark, error)
	Create(ctx context.Context, url string) (*ability.Bookmark, error)
	Delete(ctx context.Context, id string) error
	Archive(ctx context.Context, id string) (bool, error)
	Search(ctx context.Context, q *SearchQuery) (*ability.ListResult[ability.Bookmark], error)
	AttachTags(ctx context.Context, id string, tags []string) error
	DetachTags(ctx context.Context, id string, tags []string) error
	CheckURL(ctx context.Context, url string) (exists bool, id string, err error)
}
