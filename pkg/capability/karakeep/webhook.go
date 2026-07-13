package karakeep

import (
	"strings"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/capability"
	provider "github.com/flowline-io/flowbot/pkg/providers/karakeep"
	"github.com/flowline-io/flowbot/pkg/types"
)

// Webhook implements capability.WebhookConverter for Karakeep.
// It validates Bearer token auth and converts Karakeep webhook payloads.
type Webhook struct {
	getToken func() string
}

// NewWebhook creates a Webhook that reads the Bearer token from provider config
// lazily at verification time (following the example webhook pattern).
func NewWebhook() *Webhook {
	return &Webhook{
		getToken: provider.GetWebhookToken,
	}
}

// Compile-time interface check.
var _ capability.WebhookConverter = (*Webhook)(nil)

// WebhookPath returns the URL path segment for Karakeep webhooks.
// The full URL is /webhook/provider/karakeep/events.
func (*Webhook) WebhookPath() string {
	return "karakeep/events"
}

// VerifySignature validates the Bearer token from the Authorization header.
// The body parameter is accepted for interface compliance but unused
// (Bearer auth does not sign the body).
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

// Convert transforms the raw Karakeep webhook body into a DataEvent record.
// The headers parameter is accepted for interface compliance but unused.
func (*Webhook) Convert(body []byte, _ map[string]string) ([]types.DataEvent, error) {
	var payload provider.WebhookPayload
	if err := sonic.Unmarshal(body, &payload); err != nil {
		return nil, types.Errorf(types.ErrInvalidArgument, "invalid webhook payload: %v", err)
	}

	op := strings.TrimPrefix(payload.EventType, "bookmark.")

	ev := types.DataEvent{
		EventID:        types.Id(),
		EventType:      payload.EventType,
		Source:         "karakeep_webhook",
		Capability:     "karakeep",
		Operation:      op,
		EntityID:       payload.Data.Id,
		IdempotencyKey: payload.Data.Id,
		Data:           types.KV{"bookmark": toBookmark(&payload.Data), "event_type": payload.EventType},
	}
	return []types.DataEvent{ev}, nil
}
