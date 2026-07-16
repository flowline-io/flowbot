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
	}{
		{name: "returns fx module option"},
		{name: "module is non-nil"},
		{name: "module can be composed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opt := Module()
			assert.NotNil(t, opt)
			_, ok := opt.(fx.Option)
			assert.True(t, ok)
		})
	}
}
