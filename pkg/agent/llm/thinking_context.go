package llm

import "maps"

import "context"

type thinkingLevelContextKey struct{}
type assistantToolReasoningContextKey struct{}

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

// WithAssistantToolReasoning attaches reasoning_content keyed by tool_call id for prior
// assistant messages that include tool_calls.
func WithAssistantToolReasoning(ctx context.Context, reasoning map[string]string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(reasoning) == 0 {
		return ctx
	}
	copied := make(map[string]string, len(reasoning))
	maps.Copy(copied, reasoning)
	return context.WithValue(ctx, assistantToolReasoningContextKey{}, copied)
}

// AssistantToolReasoningFromContext returns tool_call id → reasoning_content, or nil when unset.
func AssistantToolReasoningFromContext(ctx context.Context) map[string]string {
	if ctx == nil {
		return nil
	}
	raw, ok := ctx.Value(assistantToolReasoningContextKey{}).(map[string]string)
	if !ok {
		return nil
	}
	return raw
}
