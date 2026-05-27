// Package memo implements the memo capability for short-form note-taking systems.
package memo

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/ability"
)

// ListQuery wraps pagination for listing memos.
type ListQuery struct {
	Page ability.PageRequest
}

// Service defines the memo capability contract.
// Provider adapters implement this interface to bridge providers and invokers.
type Service interface {
	// List returns a paginated list of memos.
	List(ctx context.Context, q *ListQuery) (*ability.ListResult[ability.Memo], error)
	// Get returns a single memo by its resource name (e.g., "memos/123").
	Get(ctx context.Context, name string) (*ability.Memo, error)
	// Create creates a new memo with the given content and visibility.
	Create(ctx context.Context, content, visibility string) (*ability.Memo, error)
	// Update updates a memo's fields identified by the update mask.
	Update(ctx context.Context, name string, data map[string]any) (*ability.Memo, error)
	// Delete removes a memo by its resource name.
	Delete(ctx context.Context, name string) error
	// HealthCheck reports whether the memo backend is reachable.
	HealthCheck(ctx context.Context) (bool, error)
	// ListRawEvents lists memos as raw events for polling support.
	ListRawEvents(ctx context.Context, cursor string) ([]any, string, error)
}
