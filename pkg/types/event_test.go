package types

import (
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDataEventCreatedAt(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		d    DataEvent
	}{
		{
			name: "default zero time serialized and restored",
			d:    DataEvent{EventID: "evt-1", EventType: "test.event"},
		},
		{
			name: "explicit created at preserved",
			d: DataEvent{
				EventID:   "evt-2",
				EventType: "test.event",
				CreatedAt: time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "round-trip serialization",
			d: DataEvent{
				EventID:   "evt-3",
				EventType: "test.event",
				CreatedAt: time.Now().Truncate(time.Second),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := sonic.Marshal(tt.d)
			require.NoError(t, err)

			var restored DataEvent
			err = sonic.Unmarshal(data, &restored)
			require.NoError(t, err)
			assert.Equal(t, tt.d.EventID, restored.EventID)
			assert.Equal(t, tt.d.EventType, restored.EventType)
			if tt.d.CreatedAt.IsZero() {
				assert.True(t, restored.CreatedAt.IsZero())
			} else {
				assert.True(t, tt.d.CreatedAt.Equal(restored.CreatedAt))
			}
		})
	}
}
