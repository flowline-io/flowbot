package types

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/require"
)

func TestDataEvent_TagsMarshaling(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		tags   KV
		hasKey bool
	}{
		{"nil tags omitted from JSON", nil, false},
		{"empty tags omitted from JSON", KV{}, false},
		{"single kv pair serializes", KV{"project": "alpha"}, true},
		{"multiple kv pairs serialize", KV{"project": "alpha", "env": "prod"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			evt := DataEvent{EventID: "evt-1", Tags: tt.tags}
			data, err := sonic.Marshal(evt)
			require.NoError(t, err)
			if tt.hasKey {
				require.Contains(t, string(data), `"tags"`)
			} else {
				require.NotContains(t, string(data), `"tags"`)
			}
		})
	}
}

func TestDataEvent_TagsRoundTrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		tags KV
	}{
		{"nil tags round-trip", nil},
		{"populated tags round-trip", KV{"project": "alpha", "env": "prod"}},
		{"single kv round-trip", KV{"project": "alpha"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			original := DataEvent{EventID: "evt-rt", Tags: tt.tags}
			data, err := sonic.Marshal(original)
			require.NoError(t, err)
			var restored DataEvent
			err = sonic.Unmarshal(data, &restored)
			require.NoError(t, err)
			require.Equal(t, original.Tags, restored.Tags)
		})
	}
}

func TestDataEvent_TagsWithData(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		data KV
		tags KV
	}{
		{"both populated", KV{"key": "val"}, KV{"tag1": "v1"}},
		{"data only without tags", KV{"key": "val"}, nil},
		{"tags only without data", nil, KV{"tag1": "v1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			evt := DataEvent{EventID: "evt-combo", Data: tt.data, Tags: tt.tags}
			raw, err := sonic.Marshal(evt)
			require.NoError(t, err)
			var restored DataEvent
			err = sonic.Unmarshal(raw, &restored)
			require.NoError(t, err)
			require.Equal(t, tt.data, restored.Data)
			require.Equal(t, tt.tags, restored.Tags)
		})
	}
}
