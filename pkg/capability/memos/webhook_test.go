package memos

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
)

func TestWebhookPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		want string
	}{
		{name: "returns memos/events path", want: "memos/events"},
		{name: "consistent path", want: "memos/events"},
		{name: "always the same", want: "memos/events"},
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
			name:    "valid token from query param",
			token:   "secret",
			headers: map[string]string{"X-Query-Token": "secret"},
			wantErr: false,
		},
		{
			name:    "empty token config rejects",
			token:   "",
			headers: map[string]string{"X-Query-Token": "secret"},
			wantErr: true,
		},
		{
			name:    "missing token query parameter",
			token:   "secret",
			headers: map[string]string{},
			wantErr: true,
		},
		{
			name:    "invalid token",
			token:   "secret",
			headers: map[string]string{"X-Query-Token": "wrong"},
			wantErr: true,
		},
		{
			name:    "Authorization Bearer alone is not accepted",
			token:   "secret",
			headers: map[string]string{"Authorization": "Bearer secret"},
			wantErr: true,
		},
		{
			name:    "token in wrong query key rejected",
			token:   "secret",
			headers: map[string]string{"X-Query-Key": "secret"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := &Webhook{getToken: func() string { return tt.token }}
			err := w.VerifySignature(tt.headers, []byte(`{}`))
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
		name           string
		body           []byte
		wantErr        bool
		wantVisibility string
	}{
		{
			name:           "valid payload with string visibility",
			body:           []byte(`{"url":"https://example.com","activityType":"memos.memo.created","creator":"users/1","memo":{"name":"memos/1","content":"Hello","visibility":"PRIVATE"}}`),
			wantErr:        false,
			wantVisibility: "PRIVATE",
		},
		{
			// Memos webhook uses encoding/json on protobuf Memo, so enums are numbers.
			name:           "numeric visibility from memos webhook protobuf json",
			body:           []byte(`{"url":"https://example.com","activityType":"memos.memo.created","creator":"users/1","memo":{"name":"memos/1","state":1,"content":"Hello","visibility":1,"property":{}}}`),
			wantErr:        false,
			wantVisibility: "PRIVATE",
		},
		{
			name:           "numeric public visibility",
			body:           []byte(`{"activityType":"memos.memo.updated","creator":"users/1","memo":{"name":"memos/2","visibility":3}}`),
			wantErr:        false,
			wantVisibility: "PUBLIC",
		},
		{
			name:    "invalid JSON",
			body:    []byte(`{invalid`),
			wantErr: true,
		},
		{
			name:    "empty body",
			body:    []byte(`{}`),
			wantErr: true,
		},
		{
			name:    "missing memo name",
			body:    []byte(`{"activityType":"memos.memo.created","memo":{"content":"test"}}`),
			wantErr: true,
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
			require.Len(t, events, 1)
			assert.NotEmpty(t, events[0].EventID)
			assert.Equal(t, "memos_webhook", events[0].Source)
			assert.Equal(t, "memos", events[0].Capability)
			if tt.wantVisibility != "" {
				memo, ok := events[0].Data["memo"].(*capability.Memo)
				require.True(t, ok)
				assert.Equal(t, tt.wantVisibility, memo.Visibility)
			}
		})
	}
}

func TestConvert_EventType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		activityType string
		wantOp       string
	}{
		{name: "created event", activityType: "memos.memo.created", wantOp: "created"},
		{name: "updated event", activityType: "memos.memo.updated", wantOp: "updated"},
		{name: "deleted event", activityType: "memos.memo.deleted", wantOp: "deleted"},
		{name: "pinned event", activityType: "memos.memo.pinned", wantOp: "pinned"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := NewWebhook()
			payload := []byte(`{"url":"https://example.com","activityType":"` + tt.activityType + `","creator":"users/1","memo":{"name":"memos/1","content":"test"}}`)
			events, err := w.Convert(payload, nil)
			require.NoError(t, err)
			require.Len(t, events, 1)
			assert.Equal(t, tt.activityType, events[0].EventType)
			assert.Equal(t, tt.wantOp, events[0].Operation)
			assert.Equal(t, "memos/1", events[0].EntityID)
			assert.Equal(t, "memos/1", events[0].IdempotencyKey)
		})
	}
}
