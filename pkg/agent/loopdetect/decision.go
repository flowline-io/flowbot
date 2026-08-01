package loopdetect

// Level is the severity of a loop-detection hit.
type Level string

const (
	// LevelNone means no loop detected.
	LevelNone Level = ""
	// LevelWarn is observational only.
	LevelWarn Level = "warn"
	// LevelCritical blocks or terminates depending on Detector.
	LevelCritical Level = "critical"
)

// DetectorName identifies which pattern fired.
type DetectorName string

const (
	// DetectorGenericRepeat is repeated identical tool+args.
	DetectorGenericRepeat DetectorName = "generic_repeat"
	// DetectorNoProgress is repeated identical tool+args+result.
	DetectorNoProgress DetectorName = "no_progress"
	// DetectorPingPong is alternating A/B calls without progress.
	DetectorPingPong DetectorName = "ping_pong"
	// DetectorGlobalCircuitBreaker is the hard no-progress cap.
	DetectorGlobalCircuitBreaker DetectorName = "global_circuit_breaker"
	// DetectorPostCompaction is the post-compaction identical-triple guard.
	DetectorPostCompaction DetectorName = "post_compaction"
)

// Decision is the outcome of Check before a tool executes.
type Decision struct {
	// Level is warn, critical, or empty when not stuck.
	Level Level
	// Detector names which pattern fired.
	Detector DetectorName
	// Count is the effective streak including the pending call when applicable.
	Count int
	// Message is a human-readable block/warn reason for logs and tool errors.
	Message string
	// Terminate requests a hard stop of the agent loop after a blocked result.
	Terminate bool
	// ArgsHash is the stable fingerprint for the pending tool+args.
	ArgsHash string
}

// Stuck reports whether any detector fired.
func (d Decision) Stuck() bool {
	return d.Level != LevelNone
}
