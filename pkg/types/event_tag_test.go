package types

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/require"
)

func TestDataEvent_TagsMarshaling(t *testing.T) {
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
