package miniflux

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/capability"
	provider "github.com/flowline-io/flowbot/pkg/providers/miniflux"
	"github.com/flowline-io/flowbot/pkg/types"
)

// Webhook implements capability.WebhookConverter for Miniflux RSS reader webhooks.
// It validates HMAC-SHA256 signatures and converts Miniflux webhook payloads
// into DataEvent records for the pipeline engine.
type Webhook struct {
	getSecret func() string
}

// NewWebhook creates a Webhook that reads the HMAC secret from the miniflux
// provider config lazily at verification time.
func NewWebhook() *Webhook {
	return &Webhook{
		getSecret: provider.GetWebhookSecret,
	}
}

// Compile-time interface check.
var _ capability.WebhookConverter = (*Webhook)(nil)

// WebhookPath returns the URL path segment for Miniflux webhooks.
// The full URL is /webhook/provider/miniflux/events.
func (*Webhook) WebhookPath() string {
	return "miniflux/events"
}

// VerifySignature validates the HMAC-SHA256 signature from the X-Miniflux-Signature header.
// The secret is read from provider config at verification time via getSecret.
func (w *Webhook) VerifySignature(headers map[string]string, body []byte) error {
	secret := w.getSecret()
	if secret == "" {
		return types.Errorf(types.ErrUnauthorized, "webhook secret not configured")
	}
	signature, ok := headers["X-Miniflux-Signature"]
	if !ok {
		return types.Errorf(types.ErrUnauthorized, "missing X-Miniflux-Signature header")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return types.Errorf(types.ErrUnauthorized, "invalid signature")
	}
	return nil
}

// Convert transforms the raw Miniflux webhook body into one or more DataEvent records.
// new_entries produces a single batch event with all entries in the Data KV.
// save_entry produces a single event with the saved entry in the Data KV.
func (*Webhook) Convert(body []byte, _ map[string]string) ([]types.DataEvent, error) {
	var payload provider.WebhookEvent
	if err := sonic.Unmarshal(body, &payload); err != nil {
		return nil, types.Errorf(types.ErrInvalidArgument, "invalid webhook payload: %v", err)
	}

	switch payload.EventType {
	case provider.NewEntriesEventType:
		entityID := ""
		if payload.Feed != nil {
			entityID = strconv.FormatInt(payload.Feed.ID, 10)
		}
		ev := types.DataEvent{
			EventID:        types.Id(),
			EventType:      types.EventReaderEntryNew,
			Source:         "miniflux_webhook",
			Capability:     "miniflux",
			EntityID:       entityID,
			IdempotencyKey: types.Id(),
			Data: types.KV{
				"feed":    payload.Feed,
				"entries": payload.Entries,
			},
		}
		return []types.DataEvent{ev}, nil

	case provider.SaveEntryEventType:
		entityID := ""
		if payload.Entry != nil {
			entityID = strconv.FormatInt(payload.Entry.ID, 10)
		}
		ev := types.DataEvent{
			EventID:        types.Id(),
			EventType:      types.EventReaderEntrySaved,
			Source:         "miniflux_webhook",
			Capability:     "miniflux",
			EntityID:       entityID,
			IdempotencyKey: entityID,
			Data: types.KV{
				"entry": payload.Entry,
			},
		}
		return []types.DataEvent{ev}, nil

	default:
		return nil, types.Errorf(types.ErrInvalidArgument, "unknown event type: %s", payload.EventType)
	}
}
