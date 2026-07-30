package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
)

func TestModule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "returns non-nil fx module option",
			run: func(t *testing.T) {
				assert.NotNil(t, Module())
			},
		},
		{
			name: "module can be composed",
			run: func(t *testing.T) {
				assert.NotNil(t, fx.Options(Module()))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t)
		})
	}
}
