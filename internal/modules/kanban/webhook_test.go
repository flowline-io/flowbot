package kanban

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWebhookConstants(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "kanban webhook id constant",
			fn: func(t *testing.T) {
				assert.Equal(t, "kanban", KanbanWebhookID)
			},
		},
		{
			name: "count",
			fn: func(t *testing.T) {
				assert.Len(t, webhookRules, 1)
			},
		},
		{
			name: "id matches constant",
			fn: func(t *testing.T) {
				assert.Equal(t, KanbanWebhookID, webhookRules[0].Id)
			},
		},
		{
			name: "secret is true",
			fn: func(t *testing.T) {
				assert.True(t, webhookRules[0].Secret)
			},
		},
		{
			name: "handler not nil",
			fn: func(t *testing.T) {
				assert.NotNil(t, webhookRules[0].Handler)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.fn(t)
		})
	}
}
