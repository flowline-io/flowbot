package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderAdapterWebhookConvert(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		payload   []byte
		result    json.RawMessage
		wantCount int
		callErr   string
		wantErr   string
	}{
		{
			name:      "converts single event",
			payload:   []byte(`{"type": "test"}`),
			result:    json.RawMessage(`[{"source": "plugin", "event_type": "test"}]`),
			wantCount: 1,
		},
		{
			name:      "converts multiple events",
			payload:   []byte(`{"type": "batch"}`),
			result:    json.RawMessage(`[{"source": "a"}, {"source": "b"}]`),
			wantCount: 2,
		},
		{
			name:      "converts empty result",
			payload:   []byte(`{}`),
			result:    json.RawMessage(`[]`),
			wantCount: 0,
		},
		{
			name:      "converts event with full fields",
			payload:   []byte(`{"type": "full"}`),
			result:    json.RawMessage(`[{"event_id": "evt-1", "event_type": "plugin.event", "source": "plugin-src", "capability": "example", "operation": "sync", "entity_id": "ent-1"}]`),
			wantCount: 1,
		},
		{
			name:    "plugin error",
			payload: []byte(`{"type": "err"}`),
			callErr: "webhook error",
			wantErr: "webhook error",
		},
		{
			name:    "invalid json from runner",
			payload: []byte(`{}`),
			result:  json.RawMessage(`not-json`),
			wantErr: "unmarshal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &stubRunner{
				callFn: func(_ context.Context, _ string, _ json.RawMessage) (json.RawMessage, error) {
					if tt.callErr != "" {
						return nil, fmt.Errorf("%s", tt.callErr)
					}
					return tt.result, nil
				},
			}
			adapter := NewProviderAdapter(runner, "test-provider")
			events, err := adapter.WebhookConvert(tt.payload)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Len(t, events, tt.wantCount)
		})
	}
}

func TestProviderAdapterWebhookConvertDefaults(t *testing.T) {
	t.Parallel()

	runner := &stubRunner{
		callFn: func(_ context.Context, _ string, _ json.RawMessage) (json.RawMessage, error) {
			return json.RawMessage(`[{"source": "test"}]`), nil
		},
	}
	adapter := NewProviderAdapter(runner, "test-provider")
	events, err := adapter.WebhookConvert([]byte(`{}`))
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.NotNil(t, events[0].Data)
	assert.NotNil(t, events[0].Tags)
	assert.False(t, events[0].CreatedAt.IsZero())
}

func TestProviderAdapterGetAuthorizeURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		state   string
		result  json.RawMessage
		wantURL string
	}{
		{
			name:    "returns authorize URL",
			state:   "state123",
			result:  json.RawMessage(`{"url": "https://example.com/oauth/authorize?state=state123"}`),
			wantURL: "https://example.com/oauth/authorize?state=state123",
		},
		{
			name:    "empty result",
			state:   "empty",
			result:  json.RawMessage(`{}`),
			wantURL: "",
		},
		{
			name:    "call error returns empty",
			state:   "error",
			result:  json.RawMessage(`{}`),
			wantURL: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &stubRunner{
				callFn: func(_ context.Context, _ string, _ json.RawMessage) (json.RawMessage, error) {
					if tt.name == "call error returns empty" {
						return nil, fmt.Errorf("connection refused")
					}
					return tt.result, nil
				},
			}
			adapter := NewProviderAdapter(runner, "test-provider")
			url := adapter.GetAuthorizeURL(tt.state)
			assert.Equal(t, tt.wantURL, url)
		})
	}
}
