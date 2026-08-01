package approval

import "fmt"

// Mode selects which tool-approval pipeline chatagent uses for interactive runs.
type Mode string

const (
	// ModeManual is DCG → full permission → ConfirmGate (Always allowed).
	ModeManual Mode = "manual"
	// ModeAuto is DCG → deny-only → flagged → aux LLM → ConfirmGate on escalate only.
	ModeAuto Mode = "auto"
	// ModeOff is DCG → deny-only → allow (no ask, no aux LLM).
	ModeOff Mode = "off"
)

// ParseMode validates and normalizes an approval mode string.
func ParseMode(raw string) (Mode, error) {
	switch Mode(raw) {
	case ModeManual, ModeAuto, ModeOff:
		return Mode(raw), nil
	case "":
		return ModeManual, nil
	default:
		return "", fmt.Errorf("approval: invalid mode %q", raw)
	}
}

// Valid reports whether m is a known mode.
func (m Mode) Valid() bool {
	switch m {
	case ModeManual, ModeAuto, ModeOff:
		return true
	default:
		return false
	}
}
