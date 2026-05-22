package ability

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/pkg/types"
)

// WebhookConverter converts a provider-specific webhook payload into DataEvent records.
// Each implementation encapsulates its own signature verification scheme.
type WebhookConverter interface {
	WebhookPath() string
	VerifySignature(headers map[string]string, body []byte) error
	Convert(body []byte, headers map[string]string) ([]types.DataEvent, error)
}

// PollingResource represents a single pollable resource type from a provider.
// Each (provider, resource) pair registers one PollingResource.
type PollingResource interface {
	ResourceName() string
	DefaultInterval() time.Duration
	DiffKey(item any) string
	ContentHash(item any) string
	CursorField() string
	List(ctx context.Context, cursor string) (PollResult, error)
}

// PollResult carries a batch of items returned by a polling List call.
type PollResult struct {
	Items      []any
	NextCursor string
	HasMore    bool
}
