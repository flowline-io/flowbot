package permission_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/agent/permission"
)

func TestPermissionKeyForToolRunCursor(t *testing.T) {
	t.Parallel()
	assert.Equal(t, permission.KeyGateway, permission.PermissionKeyForTool(permission.ToolRunCursor))
}

func TestSetGatewayDefaultAction(t *testing.T) {
	// Mutates package-level gateway default; must not call t.Parallel, and must restore
	// so other DefaultConfig readers (that do not assert KeyGateway) stay isolated.
	t.Cleanup(func() {
		permission.SetGatewayDefaultAction(permission.ActionAsk)
	})
	permission.SetGatewayDefaultAction(permission.ActionAsk)
	assert.Equal(t, permission.ActionAsk, permission.DefaultConfig()[permission.KeyGateway].Default)

	tests := []struct {
		name string
		set  permission.Action
		want permission.Action
	}{
		{name: "ask stays ask", set: permission.ActionAsk, want: permission.ActionAsk},
		{name: "allow is accepted", set: permission.ActionAllow, want: permission.ActionAllow},
		{name: "deny falls back to ask", set: permission.ActionDeny, want: permission.ActionAsk},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			permission.SetGatewayDefaultAction(tt.set)
			cfg := permission.DefaultConfig()
			assert.Equal(t, tt.want, cfg[permission.KeyGateway].Default)
		})
	}
}
