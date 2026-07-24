package core

// LegacyCapability reports whether name is a pre-CapCore capability type string.
func LegacyCapability(name string) bool {
	switch name {
	case "notify", "agent", "clip":
		return true
	default:
		return false
	}
}

// LegacyOperation maps pre-CapCore (capability, operation) to CapCore op name.
// ok is false when the pair is not a known legacy mapping.
func LegacyOperation(capability, operation string) (coreOp string, ok bool) {
	switch capability {
	case "notify":
		switch operation {
		case "send":
			return OpNotifySend, true
		case "health":
			return OpNotifyHealth, true
		}
	case "agent":
		switch operation {
		case "run":
			return OpAgentRun, true
		case "health":
			return OpAgentHealth, true
		}
	case "clip":
		switch operation {
		case "create":
			return OpClipCreate, true
		case "get":
			return OpClipGet, true
		case "health":
			return OpClipHealth, true
		}
	}
	return "", false
}
