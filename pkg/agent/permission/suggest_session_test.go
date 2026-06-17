package permission_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/permission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSuggestedPattern(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		primary string
		bash    permission.ParseBashCommand
		want    string
		ok      bool
	}{
		{name: "bash spaced", key: "bash", primary: "git diff file.go", bash: permission.ParseBashCommand{Prefix: "git diff"}, want: "git diff *", ok: true},
		{name: "bash compact", key: "bash", primary: "git status", bash: permission.ParseBashCommand{Prefix: "git status"}, want: "git status*", ok: true},
		{name: "edit parent dir", key: "edit", primary: "src/utils/tool.go", want: "src/utils/*", ok: true},
		{name: "complex bash", key: "bash", primary: "x", bash: permission.ParseBashCommand{Complex: true}, ok: false},
		{name: "bare star rejected", key: "bash", primary: "*", bash: permission.ParseBashCommand{Prefix: "*", Complex: true}, ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := permission.SuggestedPattern(tt.key, tt.primary, tt.bash)
			assert.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestSessionStateGrants(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		wantErr bool
		match   bool
	}{
		{name: "valid grant", pattern: "git status*", wantErr: false, match: true},
		{name: "broad rejected", pattern: "*", wantErr: true},
		{name: "no match", pattern: "npm *", wantErr: false, match: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := permission.NewSessionState()
			err := s.AddGrant("bash", tt.pattern)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.match, s.MatchesGrant("bash", "git status --porcelain"))
		})
	}
}

func TestDoomLoop(t *testing.T) {
	s := permission.NewSessionState()
	args := map[string]any{"command": "ls"}
	var triggered bool
	for range 3 {
		_, triggered = s.RecordDoomLoop(permission.ToolRunTerminal, args)
	}
	assert.True(t, triggered)
}
