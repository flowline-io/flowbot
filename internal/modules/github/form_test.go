package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormRules(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "should be empty"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Empty(t, formRules)
		})
	}
}
