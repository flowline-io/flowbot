package memos

import (
	"strings"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/ability"
	provider "github.com/flowline-io/flowbot/pkg/providers/memos"
	"github.com/flowline-io/flowbot/pkg/types"
)

// Webhook implements ability.WebhookConverter for Memos.
// The Memos webhook sends JSON payloads without authentication headers,
// so VerifySignature is a no-op by default.
type Webhook struct{}

// NewWebhook creates a Webhook for the Memos provider.
func NewWebhook() *Webhook {
	return &Webhook{}
}

// Compile-time interface check.
var _ ability.WebhookConverter = (*Webhook)(nil)

// WebhookPath returns the URL path segment for Memos webhooks.
// The full URL is /webhook/provider/memos/events.
func (*Webhook) WebhookPath() string {
	return "memos/events"
}

// VerifySignature validates the incoming webhook request.
// Memos does not send authentication headers, so this is a no-op
// that always succeeds. Users who need authentication should place
// a reverse proxy between Memos and the webhook receiver.
func (*Webhook) VerifySignature(_ map[string]string, _ []byte) error {
	return nil
}

// Convert transforms the raw Memos webhook body into DataEvent records.
func (*Webhook) Convert(body []byte, _ map[string]string) ([]types.DataEvent, error) {
	var payload provider.WebhookPayload
	if err := sonic.Unmarshal(body, &payload); err != nil {
		return nil, types.Errorf(types.ErrInvalidArgument, "invalid webhook payload: %v", err)
	}
	if payload.Memo.Name == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "webhook payload missing memo name")
	}

	op := strings.TrimPrefix(payload.ActivityType, "memos.memo.")

	ev := types.DataEvent{
		EventID:        types.Id(),
		EventType:      payload.ActivityType,
		Source:         "memos_webhook",
		Capability:     "memo",
		Operation:      op,
		EntityID:       payload.Memo.Name,
		IdempotencyKey: payload.Memo.Name,
		Backend:        "memos",
		Data:           types.KV{"memo": toMemo(&payload.Memo), "event_type": payload.ActivityType},
	}
	return []types.DataEvent{ev}, nil
}
