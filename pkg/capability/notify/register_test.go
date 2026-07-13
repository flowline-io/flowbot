package notify

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/hub"
)

func TestRegister(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"registers successfully"},
		{"registers with correct type"},
		{"registers with description"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Register()
			require.NoError(t, err)
			desc, ok := hub.Default.Get(hub.CapNotify)
			require.True(t, ok)
			assert.Equal(t, hub.CapNotify, desc.Type)
			assert.Equal(t, "Send notifications through the notification gateway", desc.Description)
			assert.True(t, desc.Healthy)
			assert.Empty(t, desc.App)
			assert.NotNil(t, desc.Instance)
		})
	}
}

func TestRegister_Operations(t *testing.T) {
	require.NoError(t, Register())
	desc, ok := hub.Default.Get(hub.CapNotify)
	require.True(t, ok)

	tests := []struct {
		name string
		op   string
	}{
		{"has send operation", OpSend},
		{"has digest operation", OpDigest},
		{"operations present", OpSend},
	}
	opNames := make([]string, len(desc.Operations))
	for i, op := range desc.Operations {
		opNames[i] = op.Name
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Contains(t, opNames, tt.op)
		})
	}
	assert.Len(t, desc.Operations, 2)
}

func TestRegister_SendHasInputParams(t *testing.T) {
	require.NoError(t, Register())
	desc, ok := hub.Default.Get(hub.CapNotify)
	require.True(t, ok)

	var sendOp hub.Operation
	for _, op := range desc.Operations {
		if op.Name == OpSend {
			sendOp = op
			break
		}
	}
	assert.Equal(t, OpSend, sendOp.Name)
	assert.Equal(t, "Send a notification using a template", sendOp.Description)
	assert.Len(t, sendOp.Input, 3)
	assert.Equal(t, "template_id", sendOp.Input[0].Name)
	assert.True(t, sendOp.Input[0].Required)
	assert.Equal(t, "channels", sendOp.Input[1].Name)
	assert.True(t, sendOp.Input[1].Required)
	assert.Equal(t, "payload", sendOp.Input[2].Name)
	assert.False(t, sendOp.Input[2].Required)
}
