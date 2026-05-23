package example

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/pkg/ability"
)

// ExamplePoller implements ability.PollingResource for the example provider.
// It demonstrates the polling pattern with cursor-based pagination and content hashing.
type ExamplePoller struct {
	svc     Service
	secret  []byte
	nowFunc func() time.Time
}

// NewExamplePoller creates an ExamplePoller that uses the given Service for data fetching.
func NewExamplePoller(svc Service) *ExamplePoller {
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
