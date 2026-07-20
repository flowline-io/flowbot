package karakeep

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/types"
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
			name:    "valid created payload",
			body:    []byte(`{"jobId":"1","bookmarkId":"b-1","userId":"u-1","url":"https://example.com","type":"link","operation":"created"}`),
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
			wantErr: true,
		},
		{
			name:    "missing bookmarkId",
			body:    []byte(`{"operation":"created"}`),
			wantErr: true,
		},
		{
			name:    "unknown operation still accepted",
			body:    []byte(`{"jobId":"2","bookmarkId":"b-1","operation":"custom"}`),
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
			require.Len(t, events, 1)
			assert.NotEmpty(t, events[0].EventID)
			assert.NotEmpty(t, events[0].EventType)
			assert.Equal(t, "karakeep_webhook", events[0].Source)
			assert.Equal(t, "karakeep", events[0].Capability)
		})
	}
}

func TestConvert_EventType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		operation string
		wantType  string
	}{
		{name: "created event", operation: "created", wantType: types.EventBookmarkCreated},
		{name: "edited event", operation: "edited", wantType: types.EventBookmarkUpdated},
		{name: "deleted event", operation: "deleted", wantType: types.EventBookmarkDeleted},
		{name: "crawled event", operation: "crawled", wantType: types.EventBookmarkCrawled},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := NewWebhook()
			payload := []byte(`{"jobId":"job-1","bookmarkId":"b-1","operation":"` + tt.operation + `"}`)
			events, err := w.Convert(payload, nil)
			require.NoError(t, err)
			require.Len(t, events, 1)
			assert.Equal(t, tt.wantType, events[0].EventType)
			assert.Equal(t, tt.operation, events[0].Operation)
			assert.Equal(t, "b-1", events[0].EntityID)
			assert.Equal(t, "job-1", events[0].IdempotencyKey)
		})
	}
}

func TestConvert_KarakeepNativePayload(t *testing.T) {
	t.Parallel()
	// Real Karakeep webhook body shape from apps/workers/workers/webhookWorker.ts
	tests := []struct {
		name            string
		body            string
		wantEventType   string
		wantOperation   string
		wantEntityID    string
		wantURL         string
		wantIdempotency string
		wantErr         bool
	}{
		{
			name: "created operation maps to bookmark.created",
			body: `{
				"jobId": "13653",
				"bookmarkId": "pawysmwcza03qt48gzp9fte0",
				"userId": "user-1",
				"url": "https://example.com/post",
				"type": "link",
				"operation": "created"
			}`,
			wantEventType:   types.EventBookmarkCreated,
			wantOperation:   "created",
			wantEntityID:    "pawysmwcza03qt48gzp9fte0",
			wantURL:         "https://example.com/post",
			wantIdempotency: "13653",
		},
		{
			name: "edited operation maps to bookmark.updated",
			body: `{
				"jobId": "job-2",
				"bookmarkId": "bm-2",
				"userId": "user-1",
				"url": "https://example.com/edited",
				"type": "link",
				"operation": "edited"
			}`,
			wantEventType:   types.EventBookmarkUpdated,
			wantOperation:   "edited",
			wantEntityID:    "bm-2",
			wantURL:         "https://example.com/edited",
			wantIdempotency: "job-2",
		},
		{
			name: "deleted operation maps to bookmark.deleted",
			body: `{
				"jobId": "job-3",
				"bookmarkId": "bm-3",
				"userId": "user-1",
				"operation": "deleted"
			}`,
			wantEventType:   types.EventBookmarkDeleted,
			wantOperation:   "deleted",
			wantEntityID:    "bm-3",
			wantIdempotency: "job-3",
		},
		{
			name: "crawled operation maps to bookmark.crawled",
			body: `{
				"jobId": "job-4",
				"bookmarkId": "bm-4",
				"userId": "user-1",
				"url": "https://example.com/crawled",
				"type": "link",
				"operation": "crawled"
			}`,
			wantEventType:   types.EventBookmarkCrawled,
			wantOperation:   "crawled",
			wantEntityID:    "bm-4",
			wantURL:         "https://example.com/crawled",
			wantIdempotency: "job-4",
		},
		{
			name: "ai tagged operation maps to bookmark.ai_tagged",
			body: `{
				"jobId": "job-5",
				"bookmarkId": "bm-5",
				"userId": "user-1",
				"url": "https://example.com/tagged",
				"type": "link",
				"operation": "ai tagged"
			}`,
			wantEventType:   types.EventBookmarkAITagged,
			wantOperation:   "ai tagged",
			wantEntityID:    "bm-5",
			wantURL:         "https://example.com/tagged",
			wantIdempotency: "job-5",
		},
		{
			name:    "missing operation is rejected",
			body:    `{"jobId":"job-6","bookmarkId":"bm-6"}`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := NewWebhook()
			events, err := w.Convert([]byte(tt.body), nil)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, events, 1)
			ev := events[0]
			assert.Equal(t, tt.wantEventType, ev.EventType)
			assert.NotEmpty(t, ev.EventType)
			assert.Equal(t, tt.wantOperation, ev.Operation)
			assert.Equal(t, tt.wantEntityID, ev.EntityID)
			assert.Equal(t, tt.wantIdempotency, ev.IdempotencyKey)
			assert.Equal(t, "karakeep_webhook", ev.Source)
			assert.Equal(t, "karakeep", ev.Capability)
			if tt.wantURL != "" {
				bookmark, ok := ev.Data["bookmark"].(*capability.Bookmark)
				require.True(t, ok)
				assert.Equal(t, tt.wantURL, bookmark.URL)
				assert.Equal(t, tt.wantEntityID, bookmark.ID)
			}
		})
	}
}
