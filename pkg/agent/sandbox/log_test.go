package sandbox

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSummarizeCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts RunOptions
		want string
	}{
		{
			name: "shell command",
			opts: RunOptions{Command: "echo hi"},
			want: "command=echo hi",
		},
		{
			name: "argv invocation",
			opts: RunOptions{Argv: []string{"python", "script.py"}},
			want: "argv=python script.py",
		},
		{
			name: "long command truncated",
			opts: RunOptions{Command: strings.Repeat("a", maxLoggedCommandLen+10)},
			want: "command=" + strings.Repeat("a", maxLoggedCommandLen) + "...",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, summarizeCommand(tt.opts))
		})
	}
}
