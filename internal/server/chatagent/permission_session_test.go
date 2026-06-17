package chatagent

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/permission"
	"github.com/stretchr/testify/assert"
)

func TestEffectiveConfigUsesDefaults(t *testing.T) {
	tests := []struct {
		name string
		user permission.Config
		key  string
	}{
		{name: "empty user keeps bash ask", user: permission.Config{}, key: "bash"},
		{name: "user override", user: permission.Config{"bash": {Default: permission.ActionDeny}}, key: "bash"},
		{name: "read env deny", user: permission.Config{}, key: "read"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			effective := permission.EffectiveConfig(tt.user)
			rs, ok := effective[tt.key]
			assert.True(t, ok)
			if tt.user[tt.key].Default != "" {
				assert.Equal(t, tt.user[tt.key].Default, rs.Default)
			}
		})
	}
}

func TestPermissionSessionManagerPersists(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
	}{
		{name: "same session returns same state", sessionID: "sess-a"},
		{name: "different sessions isolated", sessionID: "sess-b"},
		{name: "empty id still works", sessionID: ""},
	}
	mgr := &PermissionSessionManager{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := mgr.GetPermissionSession(tt.sessionID)
			b := mgr.GetPermissionSession(tt.sessionID)
			assert.Equal(t, a, b)
		})
	}
}
