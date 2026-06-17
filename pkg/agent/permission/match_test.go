package permission_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/permission"
	"github.com/stretchr/testify/assert"
)

func TestMatchGlob(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		value   string
		want    bool
	}{
		{name: "star suffix", pattern: "git status*", value: "git status --porcelain", want: true},
		{name: "star segment", pattern: "git *", value: "git diff", want: true},
		{name: "question mark", pattern: "a?c", value: "abc", want: true},
		{name: "no match", pattern: "npm *", value: "git diff", want: false},
		{name: "exact", pattern: "grep", value: "grep", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, permission.MatchGlob(tt.pattern, tt.value))
		})
	}
}

func TestIsOverlyBroadPattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    bool
	}{
		{name: "bare star", pattern: "*", want: true},
		{name: "scoped", pattern: "git status*", want: false},
		{name: "empty", pattern: "", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, permission.IsOverlyBroadPattern(tt.pattern))
		})
	}
}
