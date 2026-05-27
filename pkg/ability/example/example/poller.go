// Package example implements the example provider adapter for the example capability.
package example

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/pkg/ability"
	exsvc "github.com/flowline-io/flowbot/pkg/ability/example"
	"github.com/flowline-io/flowbot/pkg/types"
)

// ExamplePoller implements ability.PollingResource for the example provider.
// It polls the example provider for new and updated items via the example Service.
type ExamplePoller struct {
	svc     exsvc.Service
	secret  []byte
	nowFunc func() time.Time
}

// NewPoller creates an ExamplePoller backed by a default adapter.
func NewPoller() ability.PollingResource {
	return &ExamplePoller{
		svc:     New(),
		secret:  []byte("example-polling-secret-v1"),
		nowFunc: time.Now,
	}
}

// NewPollerWithService creates an ExamplePoller with a specific service, useful for testing.
func NewPollerWithService(svc exsvc.Service) *ExamplePoller {
	return &ExamplePoller{
		svc:     svc,
		secret:  []byte("example-polling-secret-v1"),
		nowFunc: time.Now,
	}
}

// ResourceName returns the unique name for this polling resource.
func (*ExamplePoller) ResourceName() string {
	return "example/events"
}

// DefaultInterval returns the recommended polling interval.
func (*ExamplePoller) DefaultInterval() time.Duration {
	return 60 * time.Second
}

// DiffKey returns the unique identifier for an item, used for change detection.
func (*ExamplePoller) DiffKey(item any) string {
	if m, ok := item.(map[string]any); ok {
		if id, ok := m["id"].(string); ok {
			return id
		}
	}
	return fmt.Sprintf("%v", item)
}

// ContentHash returns a SHA256 hash of the item for content-based change detection.
func (*ExamplePoller) ContentHash(item any) string {
	data := fmt.Sprintf("%v", item)
	h := sha256.New()
	_, _ = h.Write([]byte(data))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// CursorField returns the field name used for cursor-based pagination.
func (*ExamplePoller) CursorField() string {
	return "cursor"
}

// List fetches a batch of items from the provider starting after the given cursor.
func (p *ExamplePoller) List(ctx context.Context, cursor string) (ability.PollResult, error) {
	if err := ctx.Err(); err != nil {
		return ability.PollResult{}, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	items, nextCursor, err := p.svc.ListRawEvents(ctx, cursor)
	if err != nil {
		return ability.PollResult{}, err
	}
	return ability.PollResult{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    nextCursor != "",
	}, nil
}

// Compile-time check that ExamplePoller implements ability.PollingResource.
var _ ability.PollingResource = (*ExamplePoller)(nil)
