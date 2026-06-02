package web

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
)

func TestHasWebhookData(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		event *gen.DataEvent
		want  bool
	}{
		{
			name:  "nil data returns false",
			event: &gen.DataEvent{},
			want:  false,
		},
		{
			name: "has _webhook_method returns true",
			event: &gen.DataEvent{
				Data: map[string]any{"_webhook_method": "POST"},
			},
			want: true,
		},
		{
			name: "no webhook keys returns false",
			event: &gen.DataEvent{
				Data: map[string]any{"foo": "bar"},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, hasWebhookData(tt.event))
		})
	}
}

func TestGetEventStore_Nil(t *testing.T) {
	t.Parallel()
	s := getEventStore()
	assert.Nil(t, s)
}
