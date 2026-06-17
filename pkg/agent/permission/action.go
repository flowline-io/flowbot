package permission

// Action is the permission decision for one tool invocation.
type Action string

const (
	// ActionAllow runs the tool without prompting.
	ActionAllow Action = "allow"
	// ActionAsk prompts the user before running the tool.
	ActionAsk Action = "ask"
	// ActionDeny blocks the tool from running.
	ActionDeny Action = "deny"
)

// Valid reports whether a is a known permission action.
func (a Action) Valid() bool {
	switch a {
	case ActionAllow, ActionAsk, ActionDeny:
		return true
	default:
		return false
	}
}

// ParseAction converts raw text into an Action when valid.
func ParseAction(raw string) (Action, bool) {
	a := Action(raw)
	return a, a.Valid()
}

// Stricter returns the more restrictive of two actions (deny > ask > allow).
func Stricter(a, b Action) Action {
	rank := func(x Action) int {
		switch x {
		case ActionDeny:
			return 3
		case ActionAsk:
			return 2
		case ActionAllow:
			return 1
		default:
			return 0
		}
	}
	if rank(a) >= rank(b) {
		return a
	}
	return b
}
