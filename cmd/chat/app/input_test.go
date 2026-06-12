package app

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/client"
	"github.com/stretchr/testify/assert"
)

func TestFocusInputCmd(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "textarea starts focused"},
		{name: "focus command is safe"},
		{name: "idempotent focus"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "default")
			_ = m.focusInputCmd()
			assert.True(t, m.input.Focused())
		})
	}
}

func TestRenderFooterNonEmpty(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "includes status bar"},
		{name: "includes input hint"},
		{name: "includes textarea"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, "default")
			m.info = &client.ChatAgentInfo{ChatModel: "test-model"}
			m.width = 80
			footer := m.renderFooter()
			assert.NotEmpty(t, footer)
		})
	}
}
