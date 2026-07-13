package server

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/capability"
)

func TestBuildPollingState_NilDatabase(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "nil database creates polling state with nil persistence"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			state := buildPollingState()
			assert.NotNil(t, state)
		})
	}
}

// verify pollingPersistenceAdapter implements capability.Persistence.
var _ capability.Persistence = (*pollingPersistenceAdapter)(nil)
