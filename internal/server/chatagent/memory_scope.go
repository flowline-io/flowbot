package chatagent

import (
	"context"
	"fmt"
	"strings"
)

type memoryScopeKey struct{}

const updateMemoryToolName = "update_memory"

// WithMemoryScope stores the active memory scope on ctx for update_memory tool calls.
func WithMemoryScope(ctx context.Context, scope string) context.Context {
	if scope == "" {
		return ctx
	}
	return context.WithValue(ctx, memoryScopeKey{}, scope)
}

// MemoryScopeFromContext returns the memory scope stored on ctx, or empty when unset.
func MemoryScopeFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	raw, ok := ctx.Value(memoryScopeKey{}).(string)
	if !ok {
		return ""
	}
	return raw
}

// MemoryOperation extracts the operation argument from update_memory tool args.
func MemoryOperation(args map[string]any) string {
	if args == nil {
		return ""
	}
	return normalizeMemoryOperation(args["operation"])
}

func normalizeMemoryOperation(value any) string {
	if value == nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(fmt.Sprint(value)))
}
