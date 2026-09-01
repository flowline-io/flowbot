package kern

// Config configures the kern CLI runtime.
type Config struct {
	// Binary is the kern executable path; empty uses "kern" on PATH.
	Binary string
	// SecurityProfile sets --security-profile on boxes (e.g. "untrusted").
	SecurityProfile string
	// RequireLimits maps to --require-limits when true.
	RequireLimits bool
	// BindAllowed mirrors executor.mounts.bind.allowed for user bind mounts.
	BindAllowed bool
}
