package app

// RunPhase tracks whether the UI is idle, streaming, or awaiting confirmation.
type RunPhase int

const (
	PhaseIdle RunPhase = iota
	PhaseStreaming
	PhaseConfirming
	PhaseSessionPick
)

// HandleCtrlC applies Hermes-style Ctrl+C semantics for the current phase.
func HandleCtrlC(phase RunPhase) (next RunPhase, action CtrlCAction) {
	switch phase {
	case PhaseStreaming:
		return PhaseIdle, CtrlCCancelRun
	case PhaseConfirming:
		return PhaseIdle, CtrlCDenyConfirm
	case PhaseSessionPick:
		return PhaseIdle, CtrlCCancelSessionPick
	default:
		return PhaseIdle, CtrlCQuit
	}
}

// CtrlCAction describes what Ctrl+C should trigger.
type CtrlCAction int

const (
	CtrlCNone CtrlCAction = iota
	CtrlCCancelRun
	CtrlCDenyConfirm
	CtrlCCancelSessionPick
	CtrlCQuit
)

// ClearConfirmState resets confirmation UI state.
func ClearConfirmState() (pendingID, tool, summary string) {
	return "", "", ""
}
