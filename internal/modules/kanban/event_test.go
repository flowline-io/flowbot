package kanban

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestEventRules(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "count",
			fn: func(t *testing.T) {
				assert.Len(t, eventRules, 1)
			},
		},
		{
			name: "id",
			fn: func(t *testing.T) {
				assert.Equal(t, types.TaskCreateBotEventID, eventRules[0].Id)
			},
		},
		{
			name: "handler not nil",
			fn: func(t *testing.T) {
				assert.NotNil(t, eventRules[0].Handler)
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
