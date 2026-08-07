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

func TestDefaultConfigGatewayAsk(t *testing.T) {
	t.Parallel()
	permission.SetGatewayDefaultAction(permission.ActionAsk)
	cfg := permission.DefaultConfig()
	assert.Equal(t, permission.ActionAsk, cfg[permission.KeyGateway].Default)
}
