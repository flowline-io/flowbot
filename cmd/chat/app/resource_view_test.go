package app

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/client"
	"github.com/stretchr/testify/assert"
)

func TestParseSlashOpenCommand(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		wantCmd string
		wantArg string
		wantOK  bool
	}{
		{name: "open plan uri", line: "/open plan://abc", wantCmd: "open", wantArg: "plan://abc", wantOK: true},
		{name: "open file uri", line: "/open file://src/main.go", wantCmd: "open", wantArg: "file://src/main.go", wantOK: true},
		{name: "open missing arg", line: "/open", wantCmd: "open", wantArg: "", wantOK: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, arg, ok := ParseSlashCommand(tt.line)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantCmd, cmd)
			assert.Equal(t, tt.wantArg, arg)
		})
	}
}

func TestFormatResourcesHint(t *testing.T) {
	tests := []struct {
		name string
		refs int
		want string
	}{
		{name: "empty", refs: 0, want: ""},
		{name: "single", refs: 1, want: "Resources:"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var refs []client.ChatResourceRef
			if tt.refs > 0 {
				refs = []client.ChatResourceRef{{URI: "plan://p1", Title: "Plan"}}
			}
			got := formatResourcesHint(refs)
			if tt.want == "" {
				assert.Empty(t, got)
				return
			}
			assert.Contains(t, got, tt.want)
			assert.Contains(t, got, "plan://p1")
		})
	}
}
