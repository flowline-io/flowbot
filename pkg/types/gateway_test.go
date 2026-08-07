package types_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestGatewayJobTerminal(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		status types.GatewayJobStatus
		want   bool
	}{
		{name: "pending", status: types.GatewayJobPending, want: false},
		{name: "running", status: types.GatewayJobRunning, want: false},
		{name: "succeeded", status: types.GatewayJobSucceeded, want: true},
		{name: "failed", status: types.GatewayJobFailed, want: true},
		{name: "canceled", status: types.GatewayJobCanceled, want: true},
		{name: "empty", status: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, types.GatewayJobTerminal(tt.status))
		})
	}
}
