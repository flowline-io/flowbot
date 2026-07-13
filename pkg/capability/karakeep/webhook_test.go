package karakeep

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebhookPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		want string
	}{
		{name: "returns karakeep/events path", want: "karakeep/events"},
		{name: "consistent path", want: "karakeep/events"},
		{name: "always the same", want: "karakeep/events"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := NewWebhook()
			assert.Equal(t, tt.want, w.WebhookPath())
		})
	}
}

func TestVerifySignature(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		token   string
		headers map[string]string
		wantErr bool
	}{
		{
			name:    "valid Bearer token",
			token:   "secret",
			headers: map[string]string{"Authorization": "Bearer secret"},
			wantErr: false,
		},
		{
			name:    "missing Authorization header",
			token:   "secret",
			headers: map[string]string{},
			wantErr: true,
		},
		{
			name:    "wrong token",
			token:   "secret",
			headers: map[string]string{"Authorization": "Bearer wrong"},
			wantErr: true,
		},
		{
			name:    "missing Bearer prefix",
			token:   "secret",
			headers: map[string]string{"Authorization": "secret"},
			wantErr: true,
		},
		{
			name:    "empty configured token returns error",
			token:   "",
			headers: map[string]string{"Authorization": "Bearer anything"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := &Webhook{getToken: func() string { return tt.token }}
			err := w.VerifySignature(tt.headers, nil)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConvert(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		body    []byte
		wantErr bool
	}{
		{
			name:    "valid payload",
			body:    []byte(`{"event_type":"bookmark.created","data":{"id":"b-1","content":{"url":"https://example.com","type":"link"}},"timestamp":"2026-01-01T00:00:00Z"}`),
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			body:    []byte(`{invalid`),
			wantErr: true,
		},
		{
			name:    "empty body",
			body:    []byte(`{}`),
			wantErr: false,
		},
		{
			name:    "partial payload",
			body:    []byte(`{"event_type":"bookmark.updated"}`),
			wantErr: false,
		},
		{
			name:    "unknown event type",
			body:    []byte(`{"event_type":"bookmark.unknown","data":{"id":"b-1"}}`),
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
			if len(tt.body) > 2 {
				require.Len(t, events, 1)
				assert.NotEmpty(t, events[0].EventID)
				assert.Equal(t, "karakeep_webhook", events[0].Source)
				assert.Equal(t, "karakeep", events[0].Capability)
			}
		})
	}
}

func TestConvert_EventType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		eventType string
		wantOp    string
	}{
		{name: "created event", eventType: "bookmark.created", wantOp: "created"},
		{name: "updated event", eventType: "bookmark.updated", wantOp: "updated"},
		{name: "archived event", eventType: "bookmark.archived", wantOp: "archived"},
		{name: "deleted event", eventType: "bookmark.deleted", wantOp: "deleted"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := NewWebhook()
			payload := []byte(`{"event_type":"` + tt.eventType + `","data":{"id":"b-1"}}`)
			events, err := w.Convert(payload, nil)
			require.NoError(t, err)
			require.Len(t, events, 1)
			assert.Equal(t, tt.eventType, events[0].EventType)
			assert.Equal(t, tt.wantOp, events[0].Operation)
			assert.Equal(t, "b-1", events[0].EntityID)
			assert.Equal(t, "b-1", events[0].IdempotencyKey)
		})
	}
}
