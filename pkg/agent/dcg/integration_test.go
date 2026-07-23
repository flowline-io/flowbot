package dcg

import (
	"context"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegrationRealDCG(t *testing.T) {
	if _, err := exec.LookPath("dcg"); err != nil {
		t.Skip("dcg not installed")
	}

	cfgPath, err := MaterializeConfig()
	require.NoError(t, err)
	checker := NewBinaryChecker(BinaryCheckerOptions{ConfigPath: cfgPath})

	tests := []struct {
		name      string
		command   string
		wantAllow bool
	}{
		{name: "allow echo", command: "echo ok", wantAllow: true},
		{name: "deny git reset hard", command: "git reset --hard HEAD", wantAllow: false},
		{name: "deny git push force", command: "git push --force", wantAllow: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := checker.Check(context.Background(), tt.command)
			require.NoError(t, err)
			assert.Equal(t, tt.wantAllow, d.Allow)
		})
	}
}
