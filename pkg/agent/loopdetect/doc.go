// Package loopdetect detects repetitive tool-call patterns in an agent run.
//
// Detectors cover generic repeats, no-progress streaks, ping-pong alternation,
// a global circuit breaker, and a post-compaction identical-triple guard.
package loopdetect
