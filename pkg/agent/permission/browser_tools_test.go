package permission_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/permission"
	toolbrowser "github.com/flowline-io/flowbot/pkg/agent/tools/browser"
	"github.com/stretchr/testify/assert"
)

func TestPermissionKeyForBrowserTools(t *testing.T) {
	t.Parallel()
	for _, name := range toolbrowser.ActiveToolNames() {
		assert.Equal(t, permission.KeyBrowser, permission.PermissionKeyForTool(name), name)
	}
}

func TestScheduledOverlayDeniesBrowser(t *testing.T) {
	t.Parallel()
	overlay := permission.ScheduledRunOverlay()
	rule, ok := overlay[permission.KeyBrowser]
	assert.True(t, ok)
	assert.Equal(t, permission.ActionDeny, rule.Default)
}
