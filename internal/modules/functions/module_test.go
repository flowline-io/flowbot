package functions

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModuleRegister(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "register and name",
			run: func(t *testing.T) {
				require.NotPanics(t, Register)
				assert.Equal(t, "functions", Name)
			},
		},
		{
			name: "init defaults enabled",
			run: func(t *testing.T) {
				handler = moduleHandler{}
				config = configType{}
				require.NoError(t, InitForE2E(nil))
				assert.True(t, handler.IsReady())
			},
		},
		{
			name: "init can disable",
			run: func(t *testing.T) {
				handler = moduleHandler{}
				config = configType{}
				require.NoError(t, InitForE2E(json.RawMessage(`{"enabled":false}`)))
				assert.False(t, handler.IsReady())
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t)
		})
	}
}
