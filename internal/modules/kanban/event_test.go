package kanban

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestEventRules_Count(t *testing.T) {
	assert.Len(t, eventRules, 1)
}

func TestEventRules_ID(t *testing.T) {
	assert.Equal(t, types.TaskCreateBotEventID, eventRules[0].Id)
}

func TestEventRules_Handler(t *testing.T) {
	assert.NotNil(t, eventRules[0].Handler)
}
