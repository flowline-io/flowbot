package miniflux

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestWebhook_WebhookPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		want string
	}{
		{name: "returns miniflux path", want: "miniflux/events"},
		{name: "consistent path", want: "miniflux/events"},
		{name: "always same path", want: "miniflux/events"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := NewWebhook()
			assert.Equal(t, tt.want, w.WebhookPath())
		})
	}
}

func TestWebhook_VerifySignature(t *testing.T) {
	t.Parallel()
	body := []byte(`{"event_type":"new_entries","feed":{"id":8},"entries":[]}`)
	secret := "test-secret"
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
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
			headers: map[string]string{"X-Miniflux-Signature": validSig},
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
			headers: map[string]string{"X-Miniflux-Signature": "bad-signature"},
			body:    body,
			wantErr: true,
		},
		{
			name:    "empty secret returns error",
			secret:  "",
			headers: map[string]string{},
			body:    body,
			wantErr: true,
		},
		{
			name:    "wrong header name",
			secret:  secret,
			headers: map[string]string{"X-Signature": validSig},
			body:    body,
			wantErr: true,
		},
		{
			name:    "tampered body",
			secret:  secret,
			headers: map[string]string{"X-Miniflux-Signature": validSig},
			body:    []byte(`{"event_type":"tampered"}`),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := &Webhook{getSecret: func() string { return tt.secret }}
			err := w.VerifySignature(tt.headers, tt.body)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWebhook_Convert(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		body    []byte
		wantErr bool
	}{
		{
			name:    "new_entries payload",
			body:    []byte(`{"event_type":"new_entries","feed":{"id":8,"title":"Example"},"entries":[{"id":231,"title":"Article"}]}`),
			wantErr: false,
		},
		{
			name:    "save_entry payload",
			body:    []byte(`{"event_type":"save_entry","entry":{"id":592,"title":"Some article"}}`),
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			body:    []byte(`{invalid`),
			wantErr: true,
		},
		{
			name:    "unknown event type",
			body:    []byte(`{"event_type":"unknown_type"}`),
			wantErr: true,
		},
		{
			name:    "empty body",
			body:    []byte(`{}`),
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
			assert.Equal(t, "miniflux_webhook", events[0].Source)
			assert.Equal(t, "reader", events[0].Capability)
			assert.Equal(t, "miniflux", events[0].Backend)
		})
	}
}

func TestWebhook_Convert_NewEntries(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		body          []byte
		wantEventType string
		wantEntityID  string
	}{
		{
			name:          "single new entry",
			body:          []byte(`{"event_type":"new_entries","feed":{"id":8,"title":"Blog"},"entries":[{"id":231,"title":"Post"}]}`),
			wantEventType: types.EventReaderEntryNew,
			wantEntityID:  "8",
		},
		{
			name:          "multiple new entries",
			body:          []byte(`{"event_type":"new_entries","feed":{"id":8,"title":"Blog"},"entries":[{"id":231,"title":"Post 1"},{"id":232,"title":"Post 2"},{"id":233,"title":"Post 3"}]}`),
			wantEventType: types.EventReaderEntryNew,
			wantEntityID:  "8",
		},
		{
			name:          "new entries without feed",
			body:          []byte(`{"event_type":"new_entries","entries":[{"id":231,"title":"Post"}]}`),
			wantEventType: types.EventReaderEntryNew,
			wantEntityID:  "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := NewWebhook()
			events, err := w.Convert(tt.body, nil)
			require.NoError(t, err)
			require.Len(t, events, 1)
			ev := events[0]
			assert.Equal(t, tt.wantEventType, ev.EventType)
			assert.Equal(t, tt.wantEntityID, ev.EntityID)
			assert.NotEmpty(t, ev.IdempotencyKey)
			assert.Contains(t, ev.Data, "feed")
			assert.Contains(t, ev.Data, "entries")
			assert.NotNil(t, ev.Data["entries"])
		})
	}
}

func TestWebhook_Convert_SaveEntry(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		body         []byte
		wantEventType string
		wantEntityID  string
	}{
		{
			name:          "save entry with feed",
			body:          []byte(`{"event_type":"save_entry","entry":{"id":592,"title":"Article","feed":{"id":9,"title":"Blog"}}}`),
			wantEventType: types.EventReaderEntrySaved,
			wantEntityID:  "592",
		},
		{
			name:          "save entry without feed",
			body:          []byte(`{"event_type":"save_entry","entry":{"id":100,"title":"Standalone"}}`),
			wantEventType: types.EventReaderEntrySaved,
			wantEntityID:  "100",
		},
		{
			name:          "save entry with tags",
			body:          []byte(`{"event_type":"save_entry","entry":{"id":231,"title":"Tagged","tags":["tech","news"]}}`),
			wantEventType: types.EventReaderEntrySaved,
			wantEntityID:  "231",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := NewWebhook()
			events, err := w.Convert(tt.body, nil)
			require.NoError(t, err)
			require.Len(t, events, 1)
			ev := events[0]
			assert.Equal(t, tt.wantEventType, ev.EventType)
			assert.Equal(t, tt.wantEntityID, ev.EntityID)
			assert.Equal(t, tt.wantEntityID, ev.IdempotencyKey)
			assert.Contains(t, ev.Data, "entry")
		})
	}
}
