package dev

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPageRules(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "should have exactly 1 page rule",
			test: func(t *testing.T) {
				t.Parallel()
				assert.Len(t, pageRules, 1)
			},
		},
		{
			name: "should have id dev",
			test: func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, "dev", pageRules[0].Id)
			},
		},
		{
			name: "should have non-nil UI",
			test: func(t *testing.T) {
				t.Parallel()
				assert.NotNil(t, pageRules[0].UI)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}
