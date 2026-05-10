package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCronRules_Empty(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "cron rules are empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Empty(t, cronRules)
		})
	}
}
