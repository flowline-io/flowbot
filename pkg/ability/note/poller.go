package note

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/types"
)

// NotePoller implements ability.PollingResource for the note capability.
// It polls Trilium (or other note backends) for new and updated notes.
type NotePoller struct {
	svc     Service
	secret  []byte
	nowFunc func() time.Time
}

// NewNotePoller creates a NotePoller that uses the given Service for data fetching.
func NewNotePoller(svc Service) *NotePoller {
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
// Trilium's ListRawEvents returns items with a "noteId" field.
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
func (p *NotePoller) List(ctx context.Context, cursor string) (ability.PollResult, error) {
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

// Compile-time check that NotePoller implements ability.PollingResource.
var _ ability.PollingResource = (*NotePoller)(nil)
