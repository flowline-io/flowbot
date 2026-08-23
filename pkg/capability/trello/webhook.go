package trello

import (
	"fmt"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/flog"
	provider "github.com/flowline-io/flowbot/pkg/providers/trello"
	"github.com/flowline-io/flowbot/pkg/types"
)

// Webhook implements capability.WebhookConverter for Trello inbound webhooks.
type Webhook struct {
	getToken func() string
}

// NewWebhook creates a Webhook that reads the query token from provider config.
func NewWebhook() *Webhook {
	return &Webhook{getToken: provider.GetWebhookToken}
}

var (
	_ capability.WebhookConverter = (*Webhook)(nil)
	_ capability.WebhookHEADProbe = (*Webhook)(nil)
)

// WebhookPath returns the URL path segment for Trello webhooks.
// Full URL: /webhook/provider/trello/events?token=TOKEN
func (*Webhook) WebhookPath() string {
	return "trello/events"
}

// SupportsWebhookHEAD reports that Trello probes webhook URLs with HEAD during registration.
func (*Webhook) SupportsWebhookHEAD() bool {
	return true
}

// VerifySignature validates the webhook token from the query parameter.
func (w *Webhook) VerifySignature(headers map[string]string, _ []byte) error {
	return capability.VerifyQueryTokenWebhook(w.getToken, headers)
}

// Convert transforms a Trello webhook payload into DataEvent records.
func (*Webhook) Convert(body []byte, _ map[string]string) ([]types.DataEvent, error) {
	if len(body) == 0 {
		return nil, nil
	}
	var payload provider.WebhookPayload
	if err := sonic.Unmarshal(body, &payload); err != nil {
		return nil, types.Errorf(types.ErrInvalidArgument, "invalid webhook payload: %v", err)
	}
	if payload.Action.Type == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "missing action type in webhook payload")
	}
	eventType, operation := mapTrelloAction(payload.Action)
	if eventType == "" {
		flog.Warn("trello webhook: unsupported action %s", payload.Action.Type)
		return nil, nil
	}
	entityID := extractCardID(payload.Action.Data)
	if entityID == "" {
		entityID = types.Id()
	}
	ev := types.DataEvent{
		EventID:        types.Id(),
		EventType:      eventType,
		Source:         "trello_webhook",
		Capability:     provider.ID,
		Operation:      operation,
		EntityID:       entityID,
		IdempotencyKey: entityID + ":" + payload.Action.ID,
		Data: types.KV{
			"action_type": payload.Action.Type,
			"action_id":   payload.Action.ID,
			"action_data": payload.Action.Data,
		},
	}
	return []types.DataEvent{ev}, nil
}

func mapTrelloAction(action provider.WebhookAction) (eventType, operation string) {
	switch action.Type {
	case "createCard", "copyCard":
		return types.EventTrelloCardCreated, "created"
	case "updateCard":
		if listChanged(action.Data) {
			return types.EventTrelloCardMoved, "moved"
		}
		return types.EventTrelloCardUpdated, "updated"
	case "deleteCard":
		return types.EventTrelloCardDeleted, "deleted"
	default:
		return "", ""
	}
}

func listChanged(data map[string]any) bool {
	if data == nil {
		return false
	}
	before, okBefore := data["listBefore"]
	after, okAfter := data["listAfter"]
	if okBefore && okAfter {
		return fmt.Sprintf("%v", before) != fmt.Sprintf("%v", after)
	}
	return okBefore || okAfter
}

func extractCardID(data map[string]any) string {
	if data == nil {
		return ""
	}
	if card, ok := data["card"].(map[string]any); ok {
		if id, ok := card["id"].(string); ok {
			return id
		}
	}
	return ""
}
