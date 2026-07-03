package permission

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

const doomLoopThreshold = 3

const maxGrantsPerKey = 32

// SessionState holds per-session grants and doom-loop counters across runs.
type SessionState struct {
	mu         sync.Mutex
	always     map[string][]string
	doomCounts map[string]int
}

// NewSessionState creates empty session permission state.
func NewSessionState() *SessionState {
	return &SessionState{
		always:     make(map[string][]string),
		doomCounts: make(map[string]int),
	}
}

// AddGrant records an always-allow pattern for one permission key.
func (s *SessionState) AddGrant(key, pattern string) error {
	if IsOverlyBroadPattern(pattern) {
		return fmt.Errorf("pattern %q is too broad", pattern)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.always[key]) >= maxGrantsPerKey {
		return fmt.Errorf("too many grants for key %q", key)
	}
	s.always[key] = append(s.always[key], pattern)
	return nil
}

// Grants returns a copy of always-allow patterns grouped by permission key.
func (s *SessionState) Grants() map[string][]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string][]string, len(s.always))
	for key, patterns := range s.always {
		cp := append([]string(nil), patterns...)
		slices.Sort(cp)
		out[key] = cp
	}
	return out
}

// RestoreGrants replaces always-allow patterns from persisted storage.
func (s *SessionState) RestoreGrants(grants map[string][]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.always = make(map[string][]string, len(grants))
	for key, patterns := range grants {
		s.always[key] = append([]string(nil), patterns...)
	}
}

// Clear removes all session grants and doom-loop counters.
func (s *SessionState) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.always = make(map[string][]string)
	s.doomCounts = make(map[string]int)
}

// MatchesGrant reports whether input is covered by a session always grant.
func (s *SessionState) MatchesGrant(key, input string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, pattern := range s.always[key] {
		if MatchGlob(pattern, input) {
			return true
		}
	}
	return false
}

// RecordDoomLoop increments the counter for one tool invocation fingerprint.
// It returns the new count and whether the doom-loop threshold was reached.
func (s *SessionState) RecordDoomLoop(tool string, args map[string]any) (int, bool) {
	key := doomFingerprint(tool, args)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.doomCounts[key]++
	count := s.doomCounts[key]
	return count, count >= doomLoopThreshold
}

func doomFingerprint(tool string, args map[string]any) string {
	h := sha256.New()
	_, _ = h.Write([]byte(tool))
	_, _ = h.Write([]byte{0})
	keys := make([]string, 0, len(args))
	for k := range args {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		_, _ = h.Write([]byte(k))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write(fmt.Append(nil, args[k]))
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// ParentDirPattern returns a directory wildcard pattern for file paths.
func ParentDirPattern(path string) string {
	path = filepath.ToSlash(strings.TrimSpace(path))
	if path == "" || path == "." {
		return ""
	}
	dir := filepath.Dir(path)
	if dir == "." || dir == "/" {
		base := filepath.Base(path)
		if base == "." || base == "" {
			return ""
		}
		return base
	}
	return filepath.ToSlash(dir) + "/*"
}
