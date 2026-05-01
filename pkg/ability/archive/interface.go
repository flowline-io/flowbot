package archive

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/ability"
)

type AddRequest struct {
	URL       string
	Tag       string
	Depth     int
	Update    bool
	IndexOnly bool
}

type SearchQuery struct {
	Page ability.PageRequest
	Q    string
}

type Service interface {
	Add(ctx context.Context, req AddRequest) (*ability.ArchiveItem, error)
	Search(ctx context.Context, q *SearchQuery) (*ability.ListResult[ability.ArchiveItem], error)
	Get(ctx context.Context, id string) (*ability.ArchiveItem, error)
}
