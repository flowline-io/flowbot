// Package memos implements the Memos adapter for the memo capability.
package memos

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/pkg/capability"
	provider "github.com/flowline-io/flowbot/pkg/providers/memos"
	"github.com/flowline-io/flowbot/pkg/types"
)

// client defines the subset of provider.Memos methods used by this adapter.
type client interface {
	CreateMemo(ctx context.Context, content, visibility string) (*provider.Memo, error)
	GetMemo(ctx context.Context, name string) (*provider.Memo, error)
	ListMemos(ctx context.Context, params provider.ListMemosParams) (*provider.ListMemosResponse, error)
	UpdateMemo(ctx context.Context, name, content, visibility string, pinned *bool, fields []string) (*provider.Memo, error)
	DeleteMemo(ctx context.Context, name string) error
	GetCurrentUser(ctx context.Context) (*provider.User, error)
	ListRawEvents(ctx context.Context, cursor string) ([]map[string]any, string, error)
}

// Adapter implements Service using the Memos provider client.
type Adapter struct {
	client client
	now    func() time.Time
}

// New creates an Adapter using the default provider client (reads config from YAML).
// Returns nil when the provider is not configured.
func New() Service {
	if c := provider.GetClient(); c != nil {
		return NewWithClient(c)
	}
	return nil
}

// NewWithClient creates an Adapter with a specific client, useful for testing.
func NewWithClient(c client) Service {
	return &Adapter{client: c, now: time.Now}
}

// List returns a paginated list of memos.
func (a *Adapter) List(ctx context.Context, q *ListQuery) (*capability.ListResult[capability.Memo], error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if q == nil {
		q = &ListQuery{}
	}
	limit := normalizedLimit(q.Page.Limit)
	cursor := q.Page.Cursor
	params := provider.ListMemosParams{
		PageSize:  int32(limit),
		PageToken: cursor,
	}
	resp, err := a.client.ListMemos(ctx, params)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "memos list failed", err)
	}
	items := make([]*capability.Memo, len(resp.Memos))
	for i, m := range resp.Memos {
		items[i] = toMemo(&m)
	}
	return &capability.ListResult[capability.Memo]{
		Items: items,
		Page: &capability.PageInfo{
			Limit:      limit,
			HasMore:    resp.NextPageToken != "",
			NextCursor: resp.NextPageToken,
		},
	}, nil
}

// Get returns a single memo by its resource name.
func (a *Adapter) Get(ctx context.Context, name string) (*capability.Memo, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if name == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "name is required")
	}
	m, err := a.client.GetMemo(ctx, name)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "memos get failed", err)
	}
	return toMemo(m), nil
}

// Create creates a new memo with the given content and visibility.
func (a *Adapter) Create(ctx context.Context, content, visibility string) (*capability.Memo, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if content == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "content is required")
	}
	m, err := a.client.CreateMemo(ctx, content, visibility)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "memos create failed", err)
	}
	return toMemo(m), nil
}

// Update updates a memo's fields identified by the data map.
// Supported keys: content, visibility, pinned.
func (a *Adapter) Update(ctx context.Context, name string, data map[string]any) (*capability.Memo, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if name == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "name is required")
	}
	var content, visibility string
	var pinned *bool
	var fields []string
	if v, ok := data["content"].(string); ok && v != "" {
		content = v
		fields = append(fields, "content")
	}
	if v, ok := data["visibility"].(string); ok && v != "" {
		visibility = v
		fields = append(fields, "visibility")
	}
	if v, ok := data["pinned"].(bool); ok {
		pinned = &v
		fields = append(fields, "pinned")
	}
	m, err := a.client.UpdateMemo(ctx, name, content, visibility, pinned, fields)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "memos update failed", err)
	}
	return toMemo(m), nil
}

// Delete removes a memo by its resource name.
func (a *Adapter) Delete(ctx context.Context, name string) error {
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if name == "" {
		return types.Errorf(types.ErrInvalidArgument, "name is required")
	}
	if err := a.client.DeleteMemo(ctx, name); err != nil {
		return types.WrapError(types.ErrProvider, "memos delete failed", err)
	}
	return nil
}

// HealthCheck reports whether the memo backend is reachable
// by querying the current user endpoint.
func (a *Adapter) HealthCheck(ctx context.Context) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	user, err := a.client.GetCurrentUser(ctx)
	if err != nil {
		return false, types.WrapError(types.ErrProvider, "memos health check failed", err)
	}
	return user != nil, nil
}

// ListRawEvents lists memos as raw events for polling support.
func (a *Adapter) ListRawEvents(ctx context.Context, cursor string) ([]any, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	items, nextCursor, err := a.client.ListRawEvents(ctx, cursor)
	if err != nil {
		return nil, "", types.WrapError(types.ErrProvider, "memos list raw events failed", err)
	}
	result := make([]any, len(items))
	for i, item := range items {
		result[i] = item
	}
	return result, nextCursor, nil
}

// normalizedLimit clamps the provided limit to a valid range.
func normalizedLimit(limit int) int {
	const defaultLimit = 50
	if limit <= 0 || limit > provider.MaxPageSize {
		return defaultLimit
	}
	return limit
}

// toMemo maps a provider.Memo to an capability.Memo domain type.
func toMemo(m *provider.Memo) *capability.Memo {
	if m == nil {
		return nil
	}
	memo := &capability.Memo{
		Name:       m.Name,
		State:      m.State,
		Content:    m.Content,
		Visibility: m.Visibility,
		Tags:       m.Tags,
		Pinned:     m.Pinned,
		Creator:    m.Creator,
		Snippet:    m.Snippet,
	}
	if m.CreateTime != nil {
		memo.CreateTime = *m.CreateTime
	}
	if m.UpdateTime != nil {
		memo.UpdateTime = *m.UpdateTime
	}
	return memo
}
