package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWebserviceRules(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "exactly two webservice rules",
			fn: func(t *testing.T) {
				assert.Len(t, webserviceRules, 2)
			},
		},
		{
			name: "webservice rules are not empty",
			fn: func(t *testing.T) {
				assert.NotEmpty(t, webserviceRules)
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
