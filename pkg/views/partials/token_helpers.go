package partials

import (
	"strings"

	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

// tokenPrefix returns the first 12 characters of a token string plus ellipsis.
func tokenPrefix(token string) string {
	return auth.TokenPrefix(token) + "..."
}

// TokenFilterText builds the client-side filter haystack for a token row.
func TokenFilterText(item model.TokenItem) string {
	parts := []string{item.UID.String(), auth.TokenPrefix(item.Token)}
	parts = append(parts, item.Scopes...)
	return strings.Join(parts, " ")
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
	case "workflow:read":
		return "Workflow R"
	case "workflow:run":
		return "Workflow X"
	default:
		return scope
	}
}
