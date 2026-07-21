package llm

import "context"

type thinkingLevelContextKey struct{}

// WithThinkingLevel attaches a per-request thinking level to ctx for HTTP transports.
func WithThinkingLevel(ctx context.Context, level string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, thinkingLevelContextKey{}, NormalizeThinkingLevel(level))
}

// ThinkingLevelFromContext returns the thinking level stored on ctx, or default when unset.
func ThinkingLevelFromContext(ctx context.Context) string {
	if ctx == nil {
		return ThinkingLevelDefault
	}
	raw, ok := ctx.Value(thinkingLevelContextKey{}).(string)
	if !ok || raw == "" {
		return ThinkingLevelDefault
	}
	return NormalizeThinkingLevel(raw)
}
