package notify

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/hub"
)

func TestDescriptor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		wantLen int
	}{
		{"descriptor has correct type", 0},
		{"descriptor has correct description", 0},
		{"descriptor is healthy", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			desc := Descriptor()
			assert.Equal(t, hub.CapNotify, desc.Type)
			assert.Equal(t, "Send notifications through the notification gateway", desc.Description)
			assert.True(t, desc.Healthy)
			assert.Empty(t, desc.Backend)
			assert.Empty(t, desc.App)
			assert.Nil(t, desc.Instance)
		})
	}
}

func TestDescriptor_Operations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		op   string
	}{
		{"has send operation", ability.OpNotifySend},
		{"has digest operation", ability.OpNotifyDigest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			desc := Descriptor()
			opNames := make([]string, len(desc.Operations))
			for i, op := range desc.Operations {
				opNames[i] = op.Name
			}
			assert.Contains(t, opNames, tt.op)
		})
	}
	assert.Len(t, Descriptor().Operations, 2)
}

func TestDescriptor_SendHasInputParams(t *testing.T) {
	t.Parallel()
	desc := Descriptor()
	var sendOp hub.Operation
	for _, op := range desc.Operations {
		if op.Name == ability.OpNotifySend {
			sendOp = op
			break
		}
	}
	assert.Equal(t, ability.OpNotifySend, sendOp.Name)
	assert.Equal(t, "Send a notification using a template", sendOp.Description)
	assert.Len(t, sendOp.Input, 3)
	assert.Equal(t, "template_id", sendOp.Input[0].Name)
	assert.True(t, sendOp.Input[0].Required)
	assert.Equal(t, "channels", sendOp.Input[1].Name)
	assert.True(t, sendOp.Input[1].Required)
	assert.Equal(t, "payload", sendOp.Input[2].Name)
	assert.False(t, sendOp.Input[2].Required)
}
