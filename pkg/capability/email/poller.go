package email

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

// MessagePoller polls IMAP for new/updated messages.
type MessagePoller struct {
	svc     Service
	nowFunc func() time.Time
}

// NewPoller creates a MessagePoller. Returns nil when the provider is not configured.
func NewPoller() capability.PollingResource {
	svc := New()
	if svc == nil {
		return nil
	}
	return NewPollerWithService(svc)
}

// NewPollerWithService creates a MessagePoller with a specific service.
func NewPollerWithService(svc Service) *MessagePoller {
	return &MessagePoller{svc: svc, nowFunc: time.Now}
}

func (*MessagePoller) ResourceName() string { return "email/messages" }

func (*MessagePoller) DefaultInterval() time.Duration { return 60 * time.Second }

func (*MessagePoller) Capability() string { return string(hub.CapEmail) }

func (*MessagePoller) DiffKey(item any) string {
	if m, ok := item.(map[string]any); ok {
		if id, ok := m["id"].(string); ok {
			return id
		}
	}
	return fmt.Sprintf("%v", item)
}

func (*MessagePoller) ContentHash(item any) string {
	data := fmt.Sprintf("%v", item)
	h := sha256.New()
	_, _ = h.Write([]byte(data))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (*MessagePoller) CursorField() string { return "cursor" }

func (p *MessagePoller) List(ctx context.Context, cursor string) (capability.PollResult, error) {
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

// AfterEmit marks emitted messages as seen when mark_seen_after_emit is enabled.
func (p *MessagePoller) AfterEmit(ctx context.Context, items []any) error {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id, ok := m["id"].(string)
		if !ok || id == "" {
			continue
		}
		ids = append(ids, id)
	}
	return p.svc.MarkEmittedSeen(ctx, ids)
}

var (
	_ capability.PollingResource    = (*MessagePoller)(nil)
	_ capability.CapabilityProvider = (*MessagePoller)(nil)
	_ capability.PostEmitHook       = (*MessagePoller)(nil)
)
