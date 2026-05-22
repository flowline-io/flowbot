package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWorkflowExtra(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "workflow name equals workflow",
			fn: func(t *testing.T) {
				assert.Equal(t, "workflow", Name)
			},
		},
		{
			name: "all command rules have help text",
			fn: func(t *testing.T) {
				for _, r := range commandRules {
					assert.NotEmpty(t, r.Help, "command %q should have Help text", r.Define)
				}
			},
		},
		{
			name: "config type enabled is true",
			fn: func(t *testing.T) {
				cfg := configType{Enabled: true}
				assert.True(t, cfg.Enabled)
			},
		},
		{
			name: "config type enabled is false",
			fn: func(t *testing.T) {
				cfg := configType{Enabled: false}
				assert.False(t, cfg.Enabled)
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
