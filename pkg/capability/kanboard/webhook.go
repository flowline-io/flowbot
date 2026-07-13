package kanboard

import (
	"fmt"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/flog"
	provider "github.com/flowline-io/flowbot/pkg/providers/kanboard"
	"github.com/flowline-io/flowbot/pkg/types"
)

// KanboardWebhook implements capability.WebhookConverter for the Kanboard provider.
// It validates the webhook token passed as a query parameter and converts
// Kanboard webhook payloads into DataEvent records.
type KanboardWebhook struct {
	getToken func() string
}

// NewWebhook creates a KanboardWebhook that reads the webhook token
// from the kanboard provider config at verification time.
func NewWebhook() *KanboardWebhook {
	return &KanboardWebhook{
		getToken: provider.GetWebhookToken,
	}
}

var _ capability.WebhookConverter = (*KanboardWebhook)(nil)

// WebhookPath returns the URL path that receives webhook events from Kanboard.
// The full URL is /webhook/provider/kanboard/events?token=TOKEN.
func (*KanboardWebhook) WebhookPath() string {
	return "kanboard/events"
}

// VerifySignature validates the webhook token from the query parameter.
// Kanboard passes the token as a URL query string parameter ?token=TOKEN.
// The eventsource webhook handler injects query params as X-Query-* headers.
func (w *KanboardWebhook) VerifySignature(headers map[string]string, _ []byte) error {
	token := w.getToken()
	if token == "" {
		return types.Errorf(types.ErrUnauthorized, "webhook token not configured")
	}
	provided, ok := headers["X-Query-Token"]
	if !ok {
		return types.Errorf(types.ErrUnauthorized, "missing token query parameter")
	}
	if provided != token {
		return types.Errorf(types.ErrUnauthorized, "invalid token")
	}
	return nil
}

// Convert transforms the raw Kanboard webhook body into a DataEvent record.
// The Kanboard webhook payload follows the format:
//
//	{"event_name": "task.create", "event_data": {...}}
//
// Task-related events are mapped to kanban.task.* event types.
// Comment, subtask, file, and link events are not yet mapped.
func (*KanboardWebhook) Convert(body []byte, _ map[string]string) ([]types.DataEvent, error) {
	var payload provider.EventResponse
	if err := sonic.Unmarshal(body, &payload); err != nil {
		return nil, types.Errorf(types.ErrInvalidArgument, "invalid webhook payload: %v", err)
	}

	if payload.EventName == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "missing event_name in webhook payload")
	}

	eventType, operation := mapKanboardEvent(payload.EventName)
	if eventType == "" {
		flog.Warn("kanboard webhook: unsupported event %s", payload.EventName)
		return nil, nil
	}

	entityID := extractEntityID(payload.EventData)
	if entityID == "" {
		entityID = types.Id()
	}

	ev := types.DataEvent{
		EventID:        types.Id(),
		EventType:      eventType,
		Source:         "kanboard_webhook",
		Capability:     "kanboard",
		Operation:      operation,
		EntityID:       entityID,
		IdempotencyKey: entityID,
		Data: types.KV{
			"event_name": payload.EventName,
			"event_data": payload.EventData,
		},
	}
	return []types.DataEvent{ev}, nil
}

// mapKanboardEvent maps a Kanboard event name to the corresponding
// types.Event* constant and an operation string.
func mapKanboardEvent(eventName string) (eventType, operation string) {
	switch eventName {
	case provider.TaskCreateEvent:
		return types.EventKanbanTaskCreated, "created"
	case provider.TaskUpdateEvent:
		return types.EventKanbanTaskUpdated, "updated"
	case provider.TaskCloseEvent:
		return types.EventKanbanTaskCompleted, "completed"
	case provider.TaskOpenEvent:
		return types.EventKanbanTaskOpened, "opened"
	case provider.TaskMoveColumnEvent, provider.TaskMovePositionEvent,
		provider.TaskMoveSwimlaneEvent, provider.TaskMoveProjectEvent:
		return types.EventKanbanTaskMoved, "moved"
	default:
		return "", ""
	}
}

// extractEntityID extracts the task entity identifier from the webhook event data.
func extractEntityID(eventData any) string {
	data, ok := eventData.(map[string]any)
	if !ok {
		return ""
	}
	switch {
	case data["task_id"] != nil:
		return fmt.Sprintf("%v", data["task_id"])
	default:
		return ""
	}
}
