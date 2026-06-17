package permission

import (
	"os"
	"path/filepath"
	"strings"
)

// MatchGlob reports whether value matches pattern using * and ? wildcards.
func MatchGlob(pattern, value string) bool {
	pattern = expandHome(pattern)
	value = expandHome(value)
	return globWildcardMatch(pattern, value)
}

func expandHome(s string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return s
	}
	switch {
	case s == "~":
		return home
	case strings.HasPrefix(s, "~/"):
		return filepath.Join(home, strings.TrimPrefix(s, "~/"))
	case strings.HasPrefix(s, "$HOME/"):
		return filepath.Join(home, strings.TrimPrefix(s, "$HOME/"))
	case s == "$HOME":
		return home
	default:
		return s
	}
}

func globWildcardMatch(pattern, value string) bool {
	pi, vi := 0, 0
	starPi, starVi := -1, -1

	for vi < len(value) {
		if pi < len(pattern) && (pattern[pi] == value[vi] || pattern[pi] == '?') {
			pi++
			vi++
			continue
		}
		if pi < len(pattern) && pattern[pi] == '*' {
			starPi = pi
			starVi = vi
			pi++
			continue
		}
		if starPi >= 0 {
			pi = starPi + 1
			starVi++
			vi = starVi
			continue
		}
		return false
	}
	for pi < len(pattern) && pattern[pi] == '*' {
		pi++
	}
	return pi == len(pattern)
}

// IsOverlyBroadPattern reports patterns that would grant unrestricted access.
func IsOverlyBroadPattern(pattern string) bool {
	p := strings.TrimSpace(pattern)
	return p == "" || p == "*" || p == "**"
}
