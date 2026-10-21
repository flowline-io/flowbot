package route

import (
	"strings"
	"sync"

	"github.com/gofiber/fiber/v3"
)

// IdempotencyWrap claims or replays a write before the route handler runs.
// operation is the method plus the route template, for example "POST /service/kanboard/".
type IdempotencyWrap func(ctx fiber.Ctx, operation string, next fiber.Handler) error

var (
	idempotencyMu   sync.RWMutex
	idempotencyWrap IdempotencyWrap
)

// SetIdempotencyWrap installs the write idempotency hook. A nil hook leaves routes unchanged.
func SetIdempotencyWrap(fn IdempotencyWrap) {
	idempotencyMu.Lock()
	idempotencyWrap = fn
	idempotencyMu.Unlock()
}

func currentIdempotencyWrap() IdempotencyWrap {
	idempotencyMu.RLock()
	defer idempotencyMu.RUnlock()
	return idempotencyWrap
}

func wrapIdempotency(group, method, fullPath string, handler fiber.Handler) fiber.Handler {
	if !idempotencyRoute(group, method, fullPath) {
		return handler
	}
	operation := method + " " + fullPath
	return func(ctx fiber.Ctx) error {
		fn := currentIdempotencyWrap()
		if fn == nil {
			return handler(ctx)
		}
		return fn(ctx, operation, handler)
	}
}

func idempotencyRoute(group, method, fullPath string) bool {
	switch method {
	case "POST", "PUT", "PATCH", "DELETE":
	default:
		return false
	}
	if strings.Contains(fullPath, "/service/automate/functions/call") {
		return false
	}
	switch group {
	case "automate/functions", "automate/pipeline", "automate/workflow",
		"hub", "karakeep", "kanboard", "trilium", "miniflux", "gitea", "github",
		"memos", "fireflyiii", "transmission", "email", "nocodb", "trello",
		"confluence", "devops":
		return true
	default:
		return false
	}
}
