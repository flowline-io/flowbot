// Package example implements the example capability for demonstration.
package example

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/types"
)

// ListQuery wraps pagination for listing items.
type ListQuery struct {
	Page ability.PageRequest
}

// Service defines the example capability contract.
// Provider adapters implement this interface to bridge providers and invokers.
type Service interface {
	GetItem(ctx context.Context, id string) (*ability.Host, error)
	ListItems(ctx context.Context, q *ListQuery) (*ability.ListResult[ability.Host], error)
	CreateItem(ctx context.Context, title string, tags types.KV) (*ability.Host, error)
	UpdateItem(ctx context.Context, id string, data map[string]any) (*ability.Host, error)
	DeleteItem(ctx context.Context, id string) error
	HealthCheck(ctx context.Context) (bool, error)
	ListRawEvents(ctx context.Context, cursor string) ([]any, string, error)
}
