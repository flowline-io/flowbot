package partials

import (
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/pkg/auth"
)

// tokenPrefix returns the first 12 characters of a token string plus ellipsis.
func tokenPrefix(token string) string {
	return auth.TokenPrefix(token) + "..."
}

// timeSince returns a human-readable relative time string.
// Returns "never" for zero time.
func timeSince(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	d := time.Since(t)
	if d < 0 {
		return "not yet"
	}
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		m := int(d.Minutes())
		return fmt.Sprintf("%dm ago", m)
	}
	if d < 24*time.Hour {
		h := int(d.Hours())
		return fmt.Sprintf("%dh ago", h)
	}
	days := int(d.Hours() / 24)
	return fmt.Sprintf("%dd ago", days)
}

// scopeBadge returns a shortened label for a scope value.
func scopeBadge(scope string) string {
	switch scope {
	case "admin:*":
		return "Admin"
	case "pipeline:read":
		return "Pipeline R"
	case "pipeline:run":
		return "Pipeline X"
	case "workflow:run":
		return "Workflow X"
	default:
		return scope
	}
}
