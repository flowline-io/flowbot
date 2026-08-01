package loopdetect

// Config holds thresholds for tool-loop detectors.
type Config struct {
	// Enabled master-switches all detectors including post-compaction.
	Enabled bool
	// Window is the sliding history size.
	Window int
	// GenericWarn is the same tool+args count that emits a warning.
	GenericWarn int
	// GenericCritical is the same tool+args count that blocks (recoverable).
	GenericCritical int
	// NoProgressCritical blocks when identical tool+args+result streak reaches this count.
	NoProgressCritical int
	// PingPongWarn warns on alternating A/B no-progress streaks.
	PingPongWarn int
	// PingPongCritical blocks on alternating A/B no-progress streaks.
	PingPongCritical int
	// GlobalCircuitBreaker hard-stops on any no-progress streak of this length.
	GlobalCircuitBreaker int
	// PostCompactionIdentical hard-stops when the same triple repeats this many times while armed.
	PostCompactionIdentical int
	// PostCompactionWatch limits completed tool calls watched after arming.
	PostCompactionWatch int
}

// DefaultConfig returns conservative defaults from the loop-detection plan.
func DefaultConfig() Config {
	return Config{
		Enabled:                 true,
		Window:                  30,
		GenericWarn:             5,
		GenericCritical:         10,
		NoProgressCritical:      8,
		PingPongWarn:            6,
		PingPongCritical:        10,
		GlobalCircuitBreaker:    20,
		PostCompactionIdentical: 3,
		PostCompactionWatch:     10,
	}
}

func (c Config) withDefaults() Config {
	d := DefaultConfig()
	out := c
	if out.Window <= 0 {
		out.Window = d.Window
	}
	if out.GenericWarn <= 0 {
		out.GenericWarn = d.GenericWarn
	}
	if out.GenericCritical <= 0 {
		out.GenericCritical = d.GenericCritical
	}
	if out.GenericCritical < out.GenericWarn {
		out.GenericCritical = out.GenericWarn
	}
	if out.NoProgressCritical <= 0 {
		out.NoProgressCritical = d.NoProgressCritical
	}
	if out.PingPongWarn <= 0 {
		out.PingPongWarn = d.PingPongWarn
	}
	if out.PingPongCritical <= 0 {
		out.PingPongCritical = d.PingPongCritical
	}
	if out.PingPongCritical < out.PingPongWarn {
		out.PingPongCritical = out.PingPongWarn
	}
	if out.GlobalCircuitBreaker <= 0 {
		out.GlobalCircuitBreaker = d.GlobalCircuitBreaker
	}
	if out.PostCompactionIdentical <= 0 {
		out.PostCompactionIdentical = d.PostCompactionIdentical
	}
	if out.PostCompactionWatch <= 0 {
		out.PostCompactionWatch = d.PostCompactionWatch
	}
	return out
}
