package memos

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/capability"
)

// ListQuery wraps pagination for listing memos.
type ListQuery = capability.MemoListQuery

// Service defines the memo capability contract.
type Service interface {
	List(ctx context.Context, q *ListQuery) (*capability.ListResult[capability.Memo], error)
	Get(ctx context.Context, name string) (*capability.Memo, error)
	Create(ctx context.Context, content, visibility string) (*capability.Memo, error)
	Update(ctx context.Context, name string, data map[string]any) (*capability.Memo, error)
	Delete(ctx context.Context, name string) error
	HealthCheck(ctx context.Context) (bool, error)
	ListRawEvents(ctx context.Context, cursor string) ([]any, string, error)
}
