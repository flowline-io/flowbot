package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCronRules_Empty(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "cron rules are empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, cronRules)
		})
	}
}
