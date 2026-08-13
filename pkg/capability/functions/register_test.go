package functions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
)

func TestRegisterAndCatalogSpec(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		check func(t *testing.T)
	}{
		{
			name: "register ops",
			check: func(t *testing.T) {
				hub.Default.Unregister(hub.CapFunctions)
				capability.DefaultRegistry.Unregister(hub.CapFunctions, OpInvoke)
				capability.DefaultRegistry.Unregister(hub.CapFunctions, OpGet)
				capability.DefaultRegistry.Unregister(hub.CapFunctions, OpHealth)
				t.Cleanup(func() {
					hub.Default.Unregister(hub.CapFunctions)
					capability.DefaultRegistry.Unregister(hub.CapFunctions, OpInvoke)
					capability.DefaultRegistry.Unregister(hub.CapFunctions, OpGet)
					capability.DefaultRegistry.Unregister(hub.CapFunctions, OpHealth)
				})
				require.NoError(t, Register())
				spec := CatalogSpec()
				assert.Equal(t, hub.CapFunctions, spec.Type)
				require.Len(t, spec.Ops, 3)
			},
		},
		{
			name: "invoke requires version param",
			check: func(t *testing.T) {
				spec := CatalogSpec()
				var invoke *capability.OpDef
				for i := range spec.Ops {
					if spec.Ops[i].Name == OpInvoke {
						invoke = &spec.Ops[i]
						break
					}
				}
				require.NotNil(t, invoke)
				require.True(t, invoke.Mutation)
				foundVersion := false
				for _, p := range invoke.Input {
					if p.Name == "version" {
						foundVersion = true
						assert.True(t, p.Required)
					}
				}
				assert.True(t, foundVersion)
				assert.Contains(t, invoke.Description, "HTTP token/hmac")
			},
		},
		{
			name: "description documents platform auth boundary",
			check: func(t *testing.T) {
				spec := CatalogSpec()
				assert.Contains(t, spec.Description, "do not validate function HTTP secrets")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.check(t)
		})
	}
}
