package chatagent

import (
	"context"
	"strings"
)

type memoryScopeKey struct{}

// WithMemoryScope stores the active memory scope on ctx for memory tools.
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
	return strings.TrimSpace(raw)
}
