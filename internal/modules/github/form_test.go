package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormRules(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "should be empty"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, formRules)
		})
	}
}
