package loopdetect

import (
	"fmt"
	"sync"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
)

type callRecord struct {
	toolName   string
	argsHash   string
	resultHash string
}

// Detector tracks run-scoped tool-call history for loop patterns.
type Detector struct {
	cfg Config

	mu                   sync.Mutex
	history              []callRecord
	postCompactRemaining int
	postCompactHistory   []callRecord
}

// NewDetector creates a run-scoped loop detector.
func NewDetector(cfg Config) *Detector {
	return &Detector{cfg: cfg.withDefaults()}
}

// Config returns the resolved detector configuration.
func (d *Detector) Config() Config {
	if d == nil {
		return Config{}
	}
	return d.cfg
}

// Reset clears history and post-compaction arm state.
func (d *Detector) Reset() {
	if d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.history = nil
	d.postCompactRemaining = 0
	d.postCompactHistory = nil
}

// ArmPostCompaction arms the short post-compaction guard window.
func (d *Detector) ArmPostCompaction() {
	if d == nil || !d.cfg.Enabled {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.postCompactRemaining = d.cfg.PostCompactionWatch
	d.postCompactHistory = nil
}

// ResetFingerprint removes history entries for one args hash (after user allow).
func (d *Detector) ResetFingerprint(argsHash string) {
	if d == nil || argsHash == "" {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.history = filterArgsHash(d.history, argsHash)
	d.postCompactHistory = filterArgsHash(d.postCompactHistory, argsHash)
}

func filterArgsHash(in []callRecord, argsHash string) []callRecord {
	filtered := make([]callRecord, 0, len(in))
	for _, rec := range in {
		if rec.argsHash == argsHash {
			continue
		}
		filtered = append(filtered, rec)
	}
	return filtered
}

// Check evaluates pending tool+args against history before execution.
func (d *Detector) Check(toolName string, args map[string]any) Decision {
	if d == nil || !d.cfg.Enabled {
		return Decision{}
	}
	argsHash := ArgsHash(toolName, args)

	d.mu.Lock()
	defer d.mu.Unlock()

	if dec := d.checkPostCompaction(toolName, argsHash); dec.Stuck() {
		return dec
	}
	if dec := d.checkGlobal(toolName, argsHash); dec.Stuck() {
		return dec
	}
	if dec := d.checkNoProgress(toolName, argsHash); dec.Stuck() {
		return dec
	}
	if dec := d.checkPingPong(toolName, argsHash); dec.Stuck() {
		return dec
	}
	if dec := d.checkGeneric(toolName, argsHash); dec.Stuck() {
		return dec
	}
	return Decision{ArgsHash: argsHash}
}

// ObserveResult records a completed tool invocation (not blocked calls).
func (d *Detector) ObserveResult(toolName string, args map[string]any, result msg.ToolResultMessage) {
	if d == nil || !d.cfg.Enabled {
		return
	}
	rec := callRecord{
		toolName:   toolName,
		argsHash:   ArgsHash(toolName, args),
		resultHash: ResultHash(toolName, result),
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	d.history = append(d.history, rec)
	if len(d.history) > d.cfg.Window {
		d.history = append([]callRecord(nil), d.history[len(d.history)-d.cfg.Window:]...)
	}
	if d.postCompactRemaining > 0 {
		d.postCompactHistory = append(d.postCompactHistory, rec)
		d.postCompactRemaining--
	}
}

func (d *Detector) checkGlobal(toolName, argsHash string) Decision {
	count := extendingNoProgressCount(d.history, argsHash)
	if count >= d.cfg.GlobalCircuitBreaker {
		return Decision{
			Level:     LevelCritical,
			Detector:  DetectorGlobalCircuitBreaker,
			Count:     count,
			Terminate: true,
			ArgsHash:  argsHash,
			Message: fmt.Sprintf(
				"CRITICAL: %s repeated identical no-progress outcomes %d times; session blocked by global circuit breaker",
				toolName, count,
			),
		}
	}
	return Decision{}
}

func (d *Detector) checkPostCompaction(toolName, argsHash string) Decision {
	if d.postCompactRemaining <= 0 && len(d.postCompactHistory) == 0 {
		return Decision{}
	}
	// Count completed identical triples in the armed window only (not pre-arm history).
	tripleCount := identicalTripleCount(d.postCompactHistory, argsHash)
	if tripleCount >= d.cfg.PostCompactionIdentical {
		return Decision{
			Level:     LevelCritical,
			Detector:  DetectorPostCompaction,
			Count:     tripleCount,
			Terminate: true,
			ArgsHash:  argsHash,
			Message: fmt.Sprintf(
				"CRITICAL: %s repeated identical (tool,args,result) %d times after compaction; aborting",
				toolName, tripleCount,
			),
		}
	}
	return Decision{}
}

func (d *Detector) checkNoProgress(toolName, argsHash string) Decision {
	count := extendingNoProgressCount(d.history, argsHash)
	if count >= d.cfg.NoProgressCritical {
		return Decision{
			Level:    LevelCritical,
			Detector: DetectorNoProgress,
			Count:    count,
			ArgsHash: argsHash,
			Message: fmt.Sprintf(
				"BLOCKED: %s called %d times with identical arguments and results; stop retrying",
				toolName, count,
			),
		}
	}
	return Decision{}
}

func (d *Detector) checkGeneric(toolName, argsHash string) Decision {
	count := countArgs(d.history, argsHash) + 1 // include pending
	if count >= d.cfg.GenericCritical {
		return Decision{
			Level:    LevelCritical,
			Detector: DetectorGenericRepeat,
			Count:    count,
			ArgsHash: argsHash,
			Message: fmt.Sprintf(
				"BLOCKED: %s called %d times with identical arguments",
				toolName, count,
			),
		}
	}
	if count >= d.cfg.GenericWarn {
		return Decision{
			Level:    LevelWarn,
			Detector: DetectorGenericRepeat,
			Count:    count,
			ArgsHash: argsHash,
			Message: fmt.Sprintf(
				"WARN: %s repeated %d times with identical arguments",
				toolName, count,
			),
		}
	}
	return Decision{ArgsHash: argsHash}
}

func (d *Detector) checkPingPong(toolName, argsHash string) Decision {
	pp := pingPongStreak(d.history, argsHash)
	if pp.count < 2 || !pp.noProgress {
		return Decision{}
	}
	count := pp.count + 1
	if count >= d.cfg.PingPongCritical {
		return Decision{
			Level:    LevelCritical,
			Detector: DetectorPingPong,
			Count:    count,
			ArgsHash: argsHash,
			Message: fmt.Sprintf(
				"BLOCKED: ping-pong loop detected involving %s (%d alternating calls with no progress)",
				toolName, count,
			),
		}
	}
	if count >= d.cfg.PingPongWarn {
		return Decision{
			Level:    LevelWarn,
			Detector: DetectorPingPong,
			Count:    count,
			ArgsHash: argsHash,
			Message: fmt.Sprintf(
				"WARN: ping-pong pattern involving %s (%d alternating calls)",
				toolName, count,
			),
		}
	}
	return Decision{}
}

type noProgressInfo struct {
	count         int
	latestResult  string
	hasResultHash bool
}

// extendingNoProgressCount returns streak+1 when prior identical outcomes exist (pending extends them).
func extendingNoProgressCount(history []callRecord, argsHash string) int {
	streak := noProgressStreak(history, argsHash)
	if streak.count == 0 {
		return 0
	}
	return streak.count + 1
}

func noProgressStreak(history []callRecord, argsHash string) noProgressInfo {
	var info noProgressInfo
	for i := len(history) - 1; i >= 0; i-- {
		rec := history[i]
		if rec.argsHash != argsHash {
			continue
		}
		if rec.resultHash == "" {
			continue
		}
		if !info.hasResultHash {
			info.latestResult = rec.resultHash
			info.hasResultHash = true
			info.count = 1
			continue
		}
		if rec.resultHash != info.latestResult {
			break
		}
		info.count++
	}
	return info
}

func identicalTripleCount(history []callRecord, argsHash string) int {
	if len(history) == 0 {
		return 0
	}
	var latest *callRecord
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].argsHash == argsHash && history[i].resultHash != "" {
			latest = &history[i]
			break
		}
	}
	if latest == nil {
		return 0
	}
	count := 0
	for i := len(history) - 1; i >= 0; i-- {
		rec := history[i]
		if rec.argsHash == latest.argsHash && rec.resultHash == latest.resultHash && rec.toolName == latest.toolName {
			count++
			continue
		}
		if rec.argsHash == argsHash {
			break
		}
	}
	return count
}

func countArgs(history []callRecord, argsHash string) int {
	n := 0
	for _, rec := range history {
		if rec.argsHash == argsHash {
			n++
		}
	}
	return n
}

type pingPongInfo struct {
	count      int
	noProgress bool
}

func pingPongStreak(history []callRecord, pendingArgsHash string) pingPongInfo {
	if len(history) == 0 {
		return pingPongInfo{}
	}
	last := history[len(history)-1]
	if last.argsHash == pendingArgsHash {
		return pingPongInfo{}
	}
	var other string
	for i := len(history) - 2; i >= 0; i-- {
		if history[i].argsHash != last.argsHash {
			other = history[i].argsHash
			break
		}
	}
	if other == "" || other != pendingArgsHash {
		return pingPongInfo{}
	}

	alternating := 0
	for i := len(history) - 1; i >= 0; i-- {
		expected := last.argsHash
		if alternating%2 == 1 {
			expected = other
		}
		if history[i].argsHash != expected {
			break
		}
		alternating++
	}
	if alternating < 2 {
		return pingPongInfo{}
	}

	return pingPongInfo{
		count:      alternating,
		noProgress: pingPongNoProgress(history[len(history)-alternating:]),
	}
}

func pingPongNoProgress(window []callRecord) bool {
	if len(window) < 2 {
		return false
	}
	byArgs := map[string]string{}
	for _, rec := range window {
		if rec.resultHash == "" {
			return false
		}
		prev, ok := byArgs[rec.argsHash]
		if !ok {
			byArgs[rec.argsHash] = rec.resultHash
			continue
		}
		if prev != rec.resultHash {
			return false
		}
	}
	return len(byArgs) >= 2
}
