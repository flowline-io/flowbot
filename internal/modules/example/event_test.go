package example

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestEventRules(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{name: "should have exactly 1 event rule", test: func(t *testing.T) { t.Parallel(); assert.Len(t, eventRules, 1) }},
		{name: "should have ExampleBotEventID", test: func(t *testing.T) { t.Parallel(); assert.Equal(t, types.ExampleBotEventID, eventRules[0].Id) }},
		{name: "should have non-nil handler", test: func(t *testing.T) { t.Parallel(); assert.NotNil(t, eventRules[0].Handler) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}
