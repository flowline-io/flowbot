package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSlashCommand(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		wantCmd string
		wantArg string
		wantOK  bool
	}{
		{name: "help", line: "/help", wantCmd: "help", wantOK: true},
		{name: "file path", line: "/file ./main.go", wantCmd: "file", wantArg: "./main.go", wantOK: true},
		{name: "not slash", line: "hello", wantOK: false},
		{name: "status", line: "/status", wantCmd: "status", wantOK: true},
		{name: "export", line: "/export", wantCmd: "export", wantOK: true},
		{name: "export path", line: "/export ./out/chat", wantCmd: "export", wantArg: "./out/chat", wantOK: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, arg, ok := ParseSlashCommand(tt.line)
			assert.Equal(t, tt.wantOK, ok)
			if !ok {
				return
			}
			assert.Equal(t, tt.wantCmd, cmd)
			assert.Equal(t, tt.wantArg, arg)
		})
	}
}

func TestFormatFileWarning(t *testing.T) {
	tests := []struct {
		name      string
		att       FileAttachment
		wantSub   string
		wantEmpty bool
	}{
		{name: "small file", att: FileAttachment{EstTokens: 100}, wantEmpty: true},
		{name: "truncated", att: FileAttachment{Truncated: true, EstTokens: 9000}, wantSub: "[Warning]"},
		{name: "large tokens", att: FileAttachment{EstTokens: 9000}, wantSub: "Large attachment"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatFileWarning(tt.att)
			if tt.wantEmpty {
				assert.Empty(t, got)
				return
			}
			assert.Contains(t, got, tt.wantSub)
		})
	}
}

func TestEstimateTokens(t *testing.T) {
	assert.Equal(t, 250, estimateTokens(1000))
}
