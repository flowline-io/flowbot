package example

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExampleWebhook_WebhookPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		secret string
		want   string
	}{
		{name: "returns example path", secret: "test", want: "example"},
		{name: "empty secret still returns path", secret: "", want: "example"},
		{name: "consistent path", secret: "different", want: "example"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := NewExampleWebhook(tt.secret)
			assert.Equal(t, tt.want, w.WebhookPath())
		})
	}
}

func TestExampleWebhook_VerifySignature(t *testing.T) {
	t.Parallel()
	body := []byte(`{"event_type":"test.created","entity_id":"123"}`)
	secret := "test-secret"
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	validSig := hex.EncodeToString(mac.Sum(nil))

	tests := []struct {
		name    string
		secret  string
		headers map[string]string
		body    []byte
		wantErr bool
	}{
		{
			name:    "valid signature",
			secret:  secret,
			headers: map[string]string{"X-Signature": validSig},
			body:    body,
			wantErr: false,
		},
		{
			name:    "missing header",
			secret:  secret,
			headers: map[string]string{},
			body:    body,
			wantErr: true,
		},
		{
			name:    "invalid signature",
			secret:  secret,
			headers: map[string]string{"X-Signature": "bad-signature"},
			body:    body,
			wantErr: true,
		},
		{
			name:    "empty secret skips verification",
			secret:  "",
			headers: map[string]string{},
			body:    body,
			wantErr: false,
		},
		{
			name:    "wrong header name",
			secret:  secret,
			headers: map[string]string{"X-Hub-Signature": validSig},
			body:    body,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := NewExampleWebhook(tt.secret)
			err := w.VerifySignature(tt.headers, tt.body)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExampleWebhook_Convert(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		body    []byte
		wantErr bool
	}{
		{
			name:    "valid payload converts to DataEvent",
			body:    []byte(`{"event_type":"test.created","entity_id":"e-001","data":{"key":"value"}}`),
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
			body:    []byte(`{"event_type":"test.updated"}`),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := NewExampleWebhook("secret")
			events, err := w.Convert(tt.body, nil)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if len(tt.body) > 2 {
				assert.Len(t, events, 1)
				assert.NotEmpty(t, events[0].EventID)
				assert.Equal(t, "example_webhook", events[0].Source)
			}
		})
	}
}

func TestExampleWebhook_Convert_EventType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		eventType string
		entityID  string
	}{
		{name: "create event", eventType: "item.created", entityID: "e-1"},
		{name: "update event", eventType: "item.updated", entityID: "e-2"},
		{name: "delete event", eventType: "item.deleted", entityID: "e-3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := NewExampleWebhook("secret")
			payload := []byte(`{"event_type":"` + tt.eventType + `","entity_id":"` + tt.entityID + `"}`)
			events, err := w.Convert(payload, nil)
			require.NoError(t, err)
			require.Len(t, events, 1)
			assert.Equal(t, tt.eventType, events[0].EventType)
			assert.Equal(t, tt.entityID, events[0].IdempotencyKey)
		})
	}
}
