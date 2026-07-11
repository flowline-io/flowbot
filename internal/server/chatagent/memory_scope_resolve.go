package chatagent

import (
	"strings"
)

// ResolveMemoryScope picks the memory scope for one run request.
func ResolveMemoryScope(req RunRequest) string {
	if scope := strings.TrimSpace(req.MemoryScope); scope != "" {
		return scope
	}
	return "default"
}
