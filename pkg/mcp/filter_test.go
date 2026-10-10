package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatch(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		include []string
		exclude []string
		cap     string
		tool    string
		want    bool
	}{
		{name: "empty include allows", cap: "karakeep", tool: "karakeep.create", want: true},
		{name: "match does not apply inbound deny-list", include: []string{"core"}, cap: "core", tool: "core.run_terminal", want: true},
		{name: "cap include", include: []string{"karakeep"}, cap: "karakeep", tool: "karakeep.create", want: true},
		{name: "cap include misses", include: []string{"miniflux"}, cap: "karakeep", tool: "karakeep.create", want: false},
		{name: "tool include", include: []string{"karakeep.create"}, cap: "karakeep", tool: "karakeep.create", want: true},
		{name: "exclude cap", exclude: []string{"karakeep"}, cap: "karakeep", tool: "karakeep.list", want: false},
		{name: "exclude tool", exclude: []string{"karakeep.delete"}, cap: "karakeep", tool: "karakeep.delete", want: false},
		{name: "exclude other tool", exclude: []string{"karakeep.delete"}, cap: "karakeep", tool: "karakeep.list", want: true},
		{name: "remote tool without dot", include: []string{"get_state"}, cap: "ha", tool: "get_state", want: true},
		{name: "remote tool with dot", include: []string{"light.turn_on"}, cap: "ha", tool: "light.turn_on", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, Match(tt.include, tt.exclude, tt.cap, tt.tool))
		})
	}
}
