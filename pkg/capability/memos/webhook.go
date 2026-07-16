package memos

import (
	"strings"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/capability"
	provider "github.com/flowline-io/flowbot/pkg/providers/memos"
	"github.com/flowline-io/flowbot/pkg/types"
)

// Webhook implements capability.WebhookConverter for Memos.
// It validates a shared Bearer token (vendors.memos.webhook_token). Memos may not
// send auth headers natively; configure a reverse proxy or webhook gateway to attach
// Authorization: Bearer <token>, or use a Memos build that supports signing secrets
// forwarded as Bearer.
type Webhook struct {
	getToken func() string
}

// NewWebhook creates a Webhook that reads the Bearer token from provider config
// lazily at verification time.
func NewWebhook() *Webhook {
	return &Webhook{
		getToken: provider.GetWebhookToken,
	}
}

// Compile-time interface check.
var _ capability.WebhookConverter = (*Webhook)(nil)

// WebhookPath returns the URL path segment for Memos webhooks.
// The full URL is /webhook/provider/memos/events.
func (*Webhook) WebhookPath() string {
	return "memos/events"
}

// VerifySignature validates the Bearer token from the Authorization header.
// An empty webhook_token config rejects all deliveries (same as other providers).
func (w *Webhook) VerifySignature(headers map[string]string, _ []byte) error {
	token := w.getToken()
	if token == "" {
		return types.Errorf(types.ErrUnauthorized, "webhook token not configured")
	}
	auth, ok := headers["Authorization"]
	if !ok {
		return types.Errorf(types.ErrUnauthorized, "missing Authorization header")
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(auth, prefix) {
		return types.Errorf(types.ErrUnauthorized, "invalid Authorization header format")
	}
	provided := auth[len(prefix):]
	if provided != token {
		return types.Errorf(types.ErrUnauthorized, "invalid Bearer token")
	}
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
		Capability:     "memos",
		Operation:      op,
		EntityID:       payload.Memo.Name,
		IdempotencyKey: payload.Memo.Name,
		Data:           types.KV{"memo": toMemo(&payload.Memo), "event_type": payload.ActivityType},
	}
	return []types.DataEvent{ev}, nil
}
