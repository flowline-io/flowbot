package dev

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWebserviceRules(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "should have exactly 1 webservice rule",
			test: func(t *testing.T) {
				t.Parallel()
				assert.Len(t, webserviceRules, 1)
			},
		},
		{
			name: "should not be empty",
			test: func(t *testing.T) {
				t.Parallel()
				assert.NotEmpty(t, webserviceRules)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}
