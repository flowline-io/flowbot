package bookmark

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTagPrompt(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "should not be empty",
			test: func(t *testing.T) {
				assert.NotEmpty(t, tagPrompt)
			},
		},
		{
			name: "should contain required sections",
			test: func(t *testing.T) {
				assert.Contains(t, tagPrompt, "tags")
				assert.Contains(t, tagPrompt, "JSON")
				assert.Contains(t, tagPrompt, "{{.language}}")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}
