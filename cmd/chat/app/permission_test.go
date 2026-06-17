package app

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatPermissionsView(t *testing.T) {
	view := &client.ChatPermissionsView{
		Effective: map[string]any{
			"bash": map[string]any{"*": "ask"},
		},
		SessionGrants: map[string][]string{
			"bash": {"git status*"},
		},
	}
	tests := []struct {
		name    string
		scope   string
		wantSub string
		wantErr bool
	}{
		{name: "effective json", scope: "", wantSub: "bash"},
		{name: "grants scope", scope: "grants", wantSub: "git"},
		{name: "missing key", scope: "missing", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := formatPermissionsView(view, tt.scope)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Contains(t, got, tt.wantSub)
		})
	}
}
