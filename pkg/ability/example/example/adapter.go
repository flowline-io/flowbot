// Package example implements the example provider adapter for the example capability.
package example

import (
	"context"
	"strconv"
	"time"

	"github.com/flowline-io/flowbot/pkg/ability"
	exsvc "github.com/flowline-io/flowbot/pkg/ability/example"
	provider "github.com/flowline-io/flowbot/pkg/providers/example"
	"github.com/flowline-io/flowbot/pkg/types"
)

// client defines the subset of provider.Example methods used by this adapter.
type client interface {
	Get(ctx context.Context, path string) (*provider.Response, error)
	Post(ctx context.Context, path string, data any) (*provider.Response, error)
	Put(ctx context.Context, path string, data any) (*provider.Response, error)
	Delete(ctx context.Context, path string) (*provider.Response, error)
	GetStatus(ctx context.Context, code int) (*provider.Response, error)
	ListRawEvents(ctx context.Context, cursor string) ([]map[string]any, string, error)
}

// Adapter implements example.Service using the example provider client.
type Adapter struct {
	client client
	now    func() time.Time
}

// New creates an Adapter using the default provider client (reads config from YAML).
func New() exsvc.Service {
	return NewWithClient(provider.GetClient())
}

// NewWithClient creates an Adapter with a specific client, useful for testing.
func NewWithClient(c client) exsvc.Service {
	return &Adapter{
		client: c,
		now:    time.Now,
	}
}

func (a *Adapter) GetItem(ctx context.Context, id string) (*ability.Host, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if id == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	resp, err := a.client.Get(ctx, id)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "example get failed", err)
	}
	return &ability.Host{ID: id, Name: resp.Title, Status: resp.Body}, nil
}

func (a *Adapter) ListItems(ctx context.Context, _ *exsvc.ListQuery) (*ability.ListResult[ability.Host], error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	resp, err := a.client.Get(ctx, "list")
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "example list failed", err)
	}
	item := &ability.Host{ID: "item-1", Name: resp.Title, Status: "active"}
	return &ability.ListResult[ability.Host]{
		Items: []*ability.Host{item},
	}, nil
}

func (a *Adapter) CreateItem(ctx context.Context, title string, _ types.KV) (*ability.Host, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if title == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "title is required")
	}
	resp, err := a.client.Post(ctx, "create", map[string]string{"title": title})
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "example create failed", err)
	}
	return &ability.Host{ID: "created-1", Name: title, Status: strconv.Itoa(resp.ID)}, nil
}

func (a *Adapter) UpdateItem(ctx context.Context, id string, data map[string]any) (*ability.Host, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if id == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	resp, err := a.client.Put(ctx, id, data)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "example update failed", err)
	}
	return &ability.Host{ID: id, Name: resp.Title, Status: "updated"}, nil
}

func (a *Adapter) DeleteItem(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if id == "" {
		return types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	_, err := a.client.Delete(ctx, id)
	if err != nil {
		return types.WrapError(types.ErrProvider, "example delete failed", err)
	}
	return nil
}

func (a *Adapter) HealthCheck(ctx context.Context) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	_, err := a.client.GetStatus(ctx, 200)
	if err != nil {
		return false, types.WrapError(types.ErrProvider, "example health check failed", err)
	}
	return true, nil
}

func (a *Adapter) ListRawEvents(ctx context.Context, cursor string) ([]any, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	items, next, err := a.client.ListRawEvents(ctx, cursor)
	if err != nil {
		return nil, "", types.WrapError(types.ErrProvider, "example list raw events failed", err)
	}
	result := make([]any, len(items))
	for i, item := range items {
		result[i] = item
	}
	return result, next, nil
}

// NewExamplePoller creates an ExamplePoller wired with a default adapter.
func NewExamplePoller() *exsvc.ExamplePoller {
	return exsvc.NewExamplePoller(New())
}
