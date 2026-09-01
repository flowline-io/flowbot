package confluence

import (
	"strings"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/flog"
	provider "github.com/flowline-io/flowbot/pkg/providers/confluence"
	"github.com/flowline-io/flowbot/pkg/types"
)

// Webhook implements capability.WebhookConverter for Confluence automation webhooks.
// Configure Confluence Automation to POST JSON with optional fields:
//
//	{"event":"page_created","id":"evt-1","page":{"id":"123","title":"..."},"space":{"key":"DEV"}}
type Webhook struct {
	getToken func() string
}

// NewWebhook creates a Webhook that reads the query token from provider config.
func NewWebhook() *Webhook {
	return &Webhook{getToken: provider.GetWebhookToken}
}

var _ capability.WebhookConverter = (*Webhook)(nil)

// WebhookPath returns the URL path segment for Confluence webhooks.
// Full URL: /webhook/provider/confluence/events?token=TOKEN
func (*Webhook) WebhookPath() string {
	return "confluence/events"
}

// VerifySignature validates the webhook token from the query parameter.
func (w *Webhook) VerifySignature(headers map[string]string, _ []byte) error {
	return capability.VerifyQueryTokenWebhook(w.getToken, headers)
}

// Convert transforms an automation webhook payload into DataEvent records.
func (*Webhook) Convert(body []byte, _ map[string]string) ([]types.DataEvent, error) {
	if len(body) == 0 {
		return nil, nil
	}
	var payload provider.WebhookPayload
	if err := sonic.Unmarshal(body, &payload); err != nil {
		return nil, types.Errorf(types.ErrInvalidArgument, "invalid webhook payload: %v", err)
	}
	eventType, operation := mapConfluenceEvent(payload.Event)
	if eventType == "" {
		flog.Warn("confluence webhook: unsupported event %q", payload.Event)
		return nil, nil
	}
	entityID := extractPageID(payload)
	if entityID == "" {
		entityID = types.Id()
	}
	idempotencyKey := confluenceIdempotencyKey(entityID, payload)
	ev := types.DataEvent{
		EventID:        types.Id(),
		EventType:      eventType,
		Source:         "confluence_webhook",
		Capability:     provider.ID,
		Operation:      operation,
		EntityID:       entityID,
		IdempotencyKey: idempotencyKey,
		Data: types.KV{
			"event": payload.Event,
			"id":    payload.ID,
			"page":  payload.Page,
			"space": payload.Space,
		},
	}
	return []types.DataEvent{ev}, nil
}

func mapConfluenceEvent(event string) (eventType, operation string) {
	switch strings.ToLower(strings.TrimSpace(event)) {
	case "page_created":
		return types.EventConfluencePageCreated, "created"
	case "page_updated":
		return types.EventConfluencePageUpdated, "updated"
	case "page_deleted":
		return types.EventConfluencePageDeleted, "deleted"
	default:
		return "", ""
	}
}

func extractPageID(payload provider.WebhookPayload) string {
	if payload.Page != nil && payload.Page.ID != "" {
		return payload.Page.ID
	}
	return ""
}

func confluenceIdempotencyKey(entityID string, payload provider.WebhookPayload) string {
	if payload.ID != "" {
		return entityID + ":" + payload.Event + ":" + payload.ID
	}
	return entityID + ":" + payload.Event
}
