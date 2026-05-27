package kanboard

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	provider "github.com/flowline-io/flowbot/pkg/providers/kanboard"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestKanboardWebhook_WebhookPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		want string
	}{
		{name: "returns kanboard events path", want: "kanboard/events"},
		{name: "consistent path", want: "kanboard/events"},
		{name: "path does not change", want: "kanboard/events"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := NewWebhook()
			assert.Equal(t, tt.want, w.WebhookPath())
		})
	}
}

func TestKanboardWebhook_VerifySignature(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		token   string
		headers map[string]string
		wantErr bool
	}{
		{
			name:    "valid token from query param",
			token:   "test-token",
			headers: map[string]string{"X-Query-Token": "test-token"},
			wantErr: false,
		},
		{
			name:    "missing token query parameter",
			token:   "test-token",
			headers: map[string]string{},
			wantErr: true,
		},
		{
			name:    "invalid token",
			token:   "test-token",
			headers: map[string]string{"X-Query-Token": "wrong-token"},
			wantErr: true,
		},
		{
			name:    "empty configured token returns error",
			token:   "",
			headers: map[string]string{"X-Query-Token": "test-token"},
			wantErr: true,
		},
		{
			name:    "token in wrong header key",
			token:   "test-token",
			headers: map[string]string{"X-Query-Key": "test-token"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := &KanboardWebhook{getToken: func() string { return tt.token }}
			err := w.VerifySignature(tt.headers, nil)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestKanboardWebhook_Convert(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		body          []byte
		wantErr       bool
		wantEventType string
		wantOp        string
	}{
		{
			name:          "task create event",
			body:          []byte(`{"event_name":"task.create","event_data":{"task_id":5,"task":{"id":"5","title":"My task"}}}`),
			wantErr:       false,
			wantEventType: types.EventKanbanTaskCreated,
			wantOp:        "created",
		},
		{
			name:          "task update event",
			body:          []byte(`{"event_name":"task.update","event_data":{"task_id":"5","task":{"id":"5","title":"Updated task"}}}`),
			wantErr:       false,
			wantEventType: types.EventKanbanTaskUpdated,
			wantOp:        "updated",
		},
		{
			name:          "task close event",
			body:          []byte(`{"event_name":"task.close","event_data":{"task_id":5,"task":{"id":"5","title":"Completed task"}}}`),
			wantErr:       false,
			wantEventType: types.EventKanbanTaskCompleted,
			wantOp:        "completed",
		},
		{
			name:          "task open event",
			body:          []byte(`{"event_name":"task.open","event_data":{"task_id":5,"task":{"id":"5","title":"Reopened task"}}}`),
			wantErr:       false,
			wantEventType: types.EventKanbanTaskOpened,
			wantOp:        "opened",
		},
		{
			name:          "task move column event",
			body:          []byte(`{"event_name":"task.move.column","event_data":{"task_id":"4","task":{"id":"4","title":"Moved task"}}}`),
			wantErr:       false,
			wantEventType: types.EventKanbanTaskMoved,
			wantOp:        "moved",
		},
		{
			name:          "task move position event",
			body:          []byte(`{"event_name":"task.move.position","event_data":{"task_id":"4"}}`),
			wantErr:       false,
			wantEventType: types.EventKanbanTaskMoved,
			wantOp:        "moved",
		},
		{
			name:          "task move swimlane event",
			body:          []byte(`{"event_name":"task.move.swimlane","event_data":{"task_id":"4"}}`),
			wantErr:       false,
			wantEventType: types.EventKanbanTaskMoved,
			wantOp:        "moved",
		},
		{
			name:          "task move project event",
			body:          []byte(`{"event_name":"task.move.project","event_data":{"task_id":"4"}}`),
			wantErr:       false,
			wantEventType: types.EventKanbanTaskMoved,
			wantOp:        "moved",
		},
		{
			name:    "invalid JSON body",
			body:    []byte(`{invalid`),
			wantErr: true,
		},
		{
			name:    "empty body",
			body:    []byte(`{}`),
			wantErr: true,
		},
		{
			name:    "unsupported event returns nil events",
			body:    []byte(`{"event_name":"comment.create","event_data":{"task_id":"5"}}`),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := NewWebhook()
			events, err := w.Convert(tt.body, nil)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.wantEventType != "" {
				require.Len(t, events, 1)
				assert.Equal(t, tt.wantEventType, events[0].EventType)
				assert.Equal(t, tt.wantOp, events[0].Operation)
				assert.Equal(t, "kanboard_webhook", events[0].Source)
				assert.Equal(t, "kanban", events[0].Capability)
				assert.Equal(t, "kanboard", events[0].Backend)
				assert.NotEmpty(t, events[0].EventID)
				assert.NotEmpty(t, events[0].EntityID)
				assert.NotEmpty(t, events[0].IdempotencyKey)
			} else {
				assert.Empty(t, events)
			}
		})
	}
}

func TestMapKanboardEvent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		eventName string
		wantType  string
		wantOp    string
	}{
		{
			name:      "task create maps to created",
			eventName: provider.TaskCreateEvent,
			wantType:  types.EventKanbanTaskCreated,
			wantOp:    "created",
		},
		{
			name:      "task close maps to completed",
			eventName: provider.TaskCloseEvent,
			wantType:  types.EventKanbanTaskCompleted,
			wantOp:    "completed",
		},
		{
			name:      "unknown event returns empty",
			eventName: "unknown.event",
			wantType:  "",
			wantOp:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			eventType, op := mapKanboardEvent(tt.eventName)
			assert.Equal(t, tt.wantType, eventType)
			assert.Equal(t, tt.wantOp, op)
		})
	}
}

func TestExtractEntityID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		eventData any
		want      string
	}{
		{
			name:      "extracts task_id from event data",
			eventData: map[string]any{"task_id": float64(42)},
			want:      "42",
		},
		{
			name:      "empty data returns empty",
			eventData: nil,
			want:      "",
		},
		{
			name:      "non-map data returns empty",
			eventData: "not a map",
			want:      "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := extractEntityID(tt.eventData)
			assert.Equal(t, tt.want, got)
		})
	}
}
