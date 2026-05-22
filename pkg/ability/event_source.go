package ability

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/pkg/types"
)

// WebhookConverter converts a provider-specific webhook payload into DataEvent records.
// Each implementation encapsulates its own signature verification scheme.
type WebhookConverter interface {
	// WebhookPath returns the URL path that the webhook endpoint listens on.
	WebhookPath() string
	// VerifySignature validates the incoming webhook payload against the provider's signing scheme.
	VerifySignature(headers map[string]string, body []byte) error
	// Convert transforms a raw webhook payload into one or more DataEvent records.
	Convert(body []byte, headers map[string]string) ([]types.DataEvent, error)
}

// PollingResource represents a single pollable resource type from a provider.
// Each (provider, resource) pair registers one PollingResource.
type PollingResource interface {
	// ResourceName returns a unique name for the polled resource type.
	ResourceName() string
	// DefaultInterval returns the recommended polling interval for this resource.
	DefaultInterval() time.Duration
	// DiffKey returns a unique key from an item used to detect changes between polls.
	DiffKey(item any) string
	// ContentHash returns a hash of the item content for change detection.
	ContentHash(item any) string
	// CursorField returns the field name used for cursor-based pagination.
	CursorField() string
	// List fetches a batch of items from the provider starting after cursor.
	List(ctx context.Context, cursor string) (PollResult, error)
}

// PollResult carries a batch of items returned by a polling List call.
type PollResult struct {
	Items      []any
	NextCursor string
	HasMore    bool
}
