package example

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/types"
)

// ListQuery wraps pagination for listing items.
type ListQuery = capability.ExampleListQuery

// Service defines the example capability contract.
type Service interface {
	GetItem(ctx context.Context, id string) (*capability.Host, error)
	ListItems(ctx context.Context, q *ListQuery) (*capability.ListResult[capability.Host], error)
	CreateItem(ctx context.Context, title string, tags types.KV) (*capability.Host, error)
	UpdateItem(ctx context.Context, id string, data map[string]any) (*capability.Host, error)
	DeleteItem(ctx context.Context, id string) error
	HealthCheck(ctx context.Context) (bool, error)
	ListRawEvents(ctx context.Context, cursor string) ([]any, string, error)
}
