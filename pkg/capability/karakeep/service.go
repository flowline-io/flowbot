package karakeep

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/capability"
)

// ListQuery wraps pagination and filters for listing bookmarks.
type ListQuery = capability.BookmarkListQuery

// SearchQuery wraps pagination for searching bookmarks.
type SearchQuery = capability.BookmarkSearchQuery

// Service defines the bookmark capability contract.
type Service interface {
	List(ctx context.Context, q *ListQuery) (*capability.ListResult[capability.Bookmark], error)
	Get(ctx context.Context, id string) (*capability.Bookmark, error)
	Create(ctx context.Context, url string) (*capability.Bookmark, error)
	Delete(ctx context.Context, id string) error
	Archive(ctx context.Context, id string) (bool, error)
	Search(ctx context.Context, q *SearchQuery) (*capability.ListResult[capability.Bookmark], error)
	AttachTags(ctx context.Context, id string, tags []string) error
	DetachTags(ctx context.Context, id string, tags []string) error
	CheckURL(ctx context.Context, url string) (exists bool, id string, err error)
	HealthCheck(ctx context.Context) (bool, error)
}
