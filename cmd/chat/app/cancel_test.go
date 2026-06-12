package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandleCtrlC(t *testing.T) {
	tests := []struct {
		name       string
		phase      RunPhase
		wantAction CtrlCAction
		wantPhase  RunPhase
	}{
		{name: "idle quits", phase: PhaseIdle, wantAction: CtrlCQuit, wantPhase: PhaseIdle},
		{name: "streaming cancels", phase: PhaseStreaming, wantAction: CtrlCCancelRun, wantPhase: PhaseIdle},
		{name: "confirming denies", phase: PhaseConfirming, wantAction: CtrlCDenyConfirm, wantPhase: PhaseIdle},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			phase, action := HandleCtrlC(tt.phase)
			assert.Equal(t, tt.wantAction, action)
			assert.Equal(t, tt.wantPhase, phase)
		})
	}
}
