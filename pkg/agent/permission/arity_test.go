package permission_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/permission"
	"github.com/stretchr/testify/assert"
)

func TestAnalyzeBashCommand(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		wantPref    string
		wantChain   bool
		wantComplex bool
	}{
		{name: "git checkout", command: "git checkout main", wantPref: "git checkout"},
		{name: "env prefix", command: "ENV=1 git status", wantPref: "git status"},
		{name: "pipeline chain", command: "git status | grep foo", wantPref: "git status", wantChain: true},
		{name: "relative binary", command: "./bin/git status", wantPref: "git status"},
		{name: "rm arity", command: "rm -rf node_modules", wantPref: "rm"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := permission.AnalyzeBashCommand(tt.command)
			assert.Equal(t, tt.wantPref, got.Prefix)
			assert.Equal(t, tt.wantChain, got.HasChain)
			assert.Equal(t, tt.wantComplex, got.Complex)
		})
	}
}
