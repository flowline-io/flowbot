package dcg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRobotDecision(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		stdout     string
		exitCode   int
		wantAllow  bool
		wantDeny   bool
		wantReason string
		wantErr    bool
	}{
		{
			name:      "allow exit 0",
			stdout:    `{"command":"echo ok","decision":"allow"}`,
			exitCode:  0,
			wantAllow: true,
		},
		{
			name:       "deny exit 1 with reason",
			stdout:     `{"command":"rm -rf /","decision":"deny","reason":"blocks root delete","rule_id":"core.filesystem:rm-rf-root"}`,
			exitCode:   1,
			wantDeny:   true,
			wantReason: "blocks root delete",
		},
		{
			name:     "bad json fail closed",
			stdout:   `not-json`,
			exitCode: 0,
			wantErr:  true,
		},
		{
			name:     "unexpected exit code",
			stdout:   `{"decision":"allow"}`,
			exitCode: 3,
			wantErr:  true,
		},
		{
			name:     "exit 0 but deny decision",
			stdout:   `{"decision":"deny","reason":"inconsistent"}`,
			exitCode: 0,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d, err := parseRobotDecision(tt.stdout, tt.exitCode)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantAllow, d.Allow)
			assert.Equal(t, tt.wantDeny, !d.Allow)
			if tt.wantReason != "" {
				assert.Equal(t, tt.wantReason, d.Reason)
			}
		})
	}
}
