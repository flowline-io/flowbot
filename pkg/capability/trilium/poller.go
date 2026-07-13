// Package trilium implements the Trilium adapter for the note capability.
package trilium

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/types"
)

// NotePoller implements capability.PollingResource for the note capability.
// It polls Trilium for new and updated notes.
type NotePoller struct {
	svc     Service
	secret  []byte
	nowFunc func() time.Time
}

// NewPoller creates a NotePoller backed by a default adapter.
// It returns nil when the provider is not configured.
func NewPoller() capability.PollingResource {
	svc := New()
	if svc == nil {
		return nil
	}
	return &NotePoller{
		svc:     svc,
		secret:  []byte("note-polling-secret-v1"),
		nowFunc: time.Now,
	}
}

// NewPollerWithService creates a NotePoller with a specific service, useful for testing.
func NewPollerWithService(svc Service) *NotePoller {
	return &NotePoller{
		svc:     svc,
		secret:  []byte("note-polling-secret-v1"),
		nowFunc: time.Now,
	}
}

// ResourceName returns the unique name for this polling resource.
func (*NotePoller) ResourceName() string {
	return "note/events"
}

// DefaultInterval returns the recommended polling interval.
func (*NotePoller) DefaultInterval() time.Duration {
	return 120 * time.Second
}

// DiffKey returns the unique identifier for an item, used for change detection.
func (*NotePoller) DiffKey(item any) string {
	if m, ok := item.(map[string]any); ok {
		if id, ok := m["noteId"].(string); ok {
			return id
		}
	}
	return fmt.Sprintf("%v", item)
}

// ContentHash returns a SHA256 hash of the item for content-based change detection.
func (*NotePoller) ContentHash(item any) string {
	data := fmt.Sprintf("%v", item)
	h := sha256.New()
	_, _ = h.Write([]byte(data))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// CursorField returns the field name used for cursor-based pagination.
func (*NotePoller) CursorField() string {
	return "cursor"
}

// List fetches a batch of items from the provider starting after the given cursor.
func (p *NotePoller) List(ctx context.Context, cursor string) (capability.PollResult, error) {
	if err := ctx.Err(); err != nil {
		return capability.PollResult{}, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	items, nextCursor, err := p.svc.ListRawEvents(ctx, cursor)
	if err != nil {
		return capability.PollResult{}, err
	}
	return capability.PollResult{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    nextCursor != "",
	}, nil
}

// Compile-time check that NotePoller implements capability.PollingResource.
var _ capability.PollingResource = (*NotePoller)(nil)
