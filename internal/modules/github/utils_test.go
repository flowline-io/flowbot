package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeploy_FunctionExists(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "deploy function should be defined"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, deploy)
		})
	}
}
