package confluence

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/capability"
)

// ListQuery wraps pagination for list operations.
type ListQuery struct {
	Page capability.PageRequest
}

// Service defines the Confluence capability contract.
type Service interface {
	ListSpaces(ctx context.Context, q *ListQuery) (*capability.ListResult[capability.ConfluenceSpace], error)
	ListPages(ctx context.Context, spaceKey string, q *ListQuery) (*capability.ListResult[capability.ConfluencePage], error)
	GetPage(ctx context.Context, pageID string) (*capability.ConfluencePage, error)
	GetPageContent(ctx context.Context, pageID string) (string, error)
	SearchPages(ctx context.Context, cql string, q *ListQuery) (*capability.ListResult[capability.ConfluencePage], error)
	CreatePage(ctx context.Context, spaceKey, title, content string) (*capability.ConfluencePage, error)
	UpdatePage(ctx context.Context, pageID, title, content string) (*capability.ConfluencePage, error)
	DeletePage(ctx context.Context, pageID string) error
	HealthCheck(ctx context.Context) (bool, error)
}
