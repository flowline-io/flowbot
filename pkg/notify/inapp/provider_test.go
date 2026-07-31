package inapp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/notify"
	"github.com/flowline-io/flowbot/pkg/types"
)

func TestInappProvider(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "protocol and templates",
			run: func(t *testing.T) {
				var p plugin
				assert.Equal(t, ID, p.Protocol())
				assert.Contains(t, p.Templates(), "{schema}://inbox")
			},
		},
		{
			name: "send is no-op success",
			run: func(t *testing.T) {
				var p plugin
				require.NoError(t, p.Send(types.KV{}, notify.Message{Title: "t", Body: "b"}))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}
