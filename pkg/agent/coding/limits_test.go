package coding_test

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/coding"
	"github.com/stretchr/testify/assert"
)

func TestClampMaxMatches(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		arg        int
		defaultMax int
		hardMax    int
		want       int
	}{
		{name: "uses default when unset", arg: 0, defaultMax: 200, hardMax: 1000, want: 200},
		{name: "keeps explicit under hard max", arg: 50, defaultMax: 200, hardMax: 1000, want: 50},
		{name: "clamps above hard max", arg: 5000, defaultMax: 200, hardMax: 1000, want: 1000},
		{name: "negative uses default", arg: -1, defaultMax: 100, hardMax: 500, want: 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, coding.ClampMaxMatches(tt.arg, tt.defaultMax, tt.hardMax))
		})
	}
}

func TestShouldSkipDir(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		dir  string
		want bool
	}{
		{name: "skips git", dir: ".git", want: true},
		{name: "skips node_modules", dir: "node_modules", want: true},
		{name: "keeps src", dir: "src", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, coding.ShouldSkipDir(tt.dir))
		})
	}
}

func TestDefaultLimitConstants(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		got  int
		want int
	}{
		{name: "default max output", got: coding.DefaultMaxOutput, want: 8192},
		{name: "max list dir entries", got: coding.MaxListDirEntries, want: 500},
		{name: "hard glob max", got: coding.HardGlobMaxMatches, want: 1000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.got)
		})
	}
}
