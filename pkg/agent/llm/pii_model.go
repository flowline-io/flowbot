package llm

import (
	"context"
	"strings"
	"sync"

	"github.com/tmc/langchaingo/llms"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/metrics"
)

// piiModel wraps an llms.Model and optionally anonymizes outbound text.
// Enabled is checked on every GenerateContent so config hot-reload stays correct
// while models remain pooled.
type piiModel struct {
	inner    llms.Model
	analyzer *analyzerClient
}

func wrapPIIModel(inner llms.Model) llms.Model {
	if inner == nil {
		return nil
	}
	return &piiModel{inner: inner, analyzer: newAnalyzerClientFromConfig()}
}

// Call implements llms.Model.
func (m *piiModel) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	resp, err := m.GenerateContent(ctx, []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}, options...)
	if err != nil {
		return "", err
	}
	if resp == nil || len(resp.Choices) == 0 {
		return "", nil
	}
	return resp.Choices[0].Content, nil
}

// GenerateContent anonymizes outbound message text when pii.enabled, then restores
// streamed deltas, final content, and tool-call arguments.
func (m *piiModel) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	if !config.App.PII.Enabled {
		metrics.Agent().IncPII("disabled")
		return m.inner.GenerateContent(ctx, messages, options...)
	}

	table := tableFromContext(ctx)
	analyzer := m.analyzer
	if analyzer == nil || analyzer.baseURL == "" {
		analyzer = newAnalyzerClientFromConfig()
		m.analyzer = analyzer
	}

	anonMessages, err := anonymizeMessages(ctx, analyzer, table, messages)
	if err != nil {
		metrics.Agent().IncPII("error")
		return nil, err
	}

	restorer := newStreamRestorer(table)
	callOpts, flushStream := wrapPIIStreamOptions(ctx, options, restorer)

	resp, err := m.inner.GenerateContent(ctx, anonMessages, callOpts...)
	if err != nil {
		metrics.Agent().IncPII("error")
		return nil, err
	}
	if err := flushStream(); err != nil {
		metrics.Agent().IncPII("error")
		return nil, err
	}
	restoreResponse(table, resp)
	metrics.Agent().IncPII("ok")
	return resp, nil
}

func anonymizeMessages(ctx context.Context, analyzer *analyzerClient, table *piiTable, messages []llms.MessageContent) ([]llms.MessageContent, error) {
	out := make([]llms.MessageContent, len(messages))
	for i, msg := range messages {
		out[i] = llms.MessageContent{Role: msg.Role, Parts: make([]llms.ContentPart, len(msg.Parts))}
		copy(out[i].Parts, msg.Parts)
		if msg.Role == llms.ChatMessageTypeSystem {
			continue
		}
		for j, part := range msg.Parts {
			text, ok := part.(llms.TextContent)
			if !ok || text.Text == "" {
				continue
			}
			anon, err := anonymizeText(ctx, analyzer, table, text.Text)
			if err != nil {
				return nil, err
			}
			out[i].Parts[j] = llms.TextContent{Text: anon}
		}
	}
	return out, nil
}

func anonymizeText(ctx context.Context, analyzer *analyzerClient, table *piiTable, text string) (string, error) {
	table.mu.Lock()
	sum := hashText(text)
	if cached, ok := table.hashCache[sum]; ok {
		table.touch()
		table.mu.Unlock()
		return cached, nil
	}
	table.mu.Unlock()

	spans, err := analyzer.analyzeDual(ctx, text)
	if err != nil {
		return "", err
	}

	table.mu.Lock()
	defer table.mu.Unlock()
	if cached, ok := table.hashCache[sum]; ok {
		table.touch()
		return cached, nil
	}
	return table.anonymizeLocked(text, spans), nil
}

func restoreResponse(table *piiTable, resp *llms.ContentResponse) {
	if resp == nil || table == nil {
		return
	}
	table.mu.Lock()
	defer table.mu.Unlock()
	for _, choice := range resp.Choices {
		if choice == nil {
			continue
		}
		choice.Content = table.restoreLocked(choice.Content)
		for j := range choice.ToolCalls {
			tc := &choice.ToolCalls[j]
			if tc.FunctionCall != nil {
				tc.FunctionCall.Arguments = table.restoreLocked(tc.FunctionCall.Arguments)
			}
		}
	}
}

func wrapPIIStreamOptions(ctx context.Context, options []llms.CallOption, restorer *streamRestorer) ([]llms.CallOption, func() error) {
	var opts llms.CallOptions
	for _, o := range options {
		if o != nil {
			o(&opts)
		}
	}
	out := append([]llms.CallOption{}, options...)
	noopFlush := func() error { return nil }

	if opts.StreamingReasoningFunc != nil {
		inner := opts.StreamingReasoningFunc
		out = append(out, llms.WithStreamingReasoningFunc(func(streamCtx context.Context, reasoningChunk, chunk []byte) error {
			var restoredReasoning []byte
			if len(reasoningChunk) > 0 {
				restoredReasoning = []byte(restorer.push(string(reasoningChunk)))
			}
			var restoredChunk []byte
			if len(chunk) > 0 {
				restoredChunk = []byte(restorer.push(string(chunk)))
			}
			if len(restoredReasoning) == 0 && len(restoredChunk) == 0 {
				return nil
			}
			return inner(streamCtx, restoredReasoning, restoredChunk)
		}))
		return out, func() error {
			rem := restorer.flush()
			if rem == "" {
				return nil
			}
			return inner(ctx, nil, []byte(rem))
		}
	}
	if opts.StreamingFunc != nil {
		inner := opts.StreamingFunc
		out = append(out, llms.WithStreamingFunc(func(streamCtx context.Context, chunk []byte) error {
			if len(chunk) == 0 {
				return nil
			}
			restored := restorer.push(string(chunk))
			if restored == "" {
				return nil
			}
			return inner(streamCtx, []byte(restored))
		}))
		return out, func() error {
			rem := restorer.flush()
			if rem == "" {
				return nil
			}
			return inner(ctx, []byte(rem))
		}
	}
	return out, noopFlush
}

// streamRestorer buffers incomplete {{PII_*_*}} tokens across stream chunks.
type streamRestorer struct {
	table *piiTable
	mu    sync.Mutex
	buf   strings.Builder
}

func newStreamRestorer(table *piiTable) *streamRestorer {
	return &streamRestorer{table: table}
}

func (r *streamRestorer) push(delta string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if delta == "" {
		return ""
	}
	_, _ = r.buf.WriteString(delta)
	raw := r.buf.String()
	emit, keep := splitRestorable(raw)
	r.buf.Reset()
	if keep != "" {
		_, _ = r.buf.WriteString(keep)
	}
	if emit == "" {
		return ""
	}
	r.table.mu.Lock()
	defer r.table.mu.Unlock()
	return r.table.restoreLocked(emit)
}

func (r *streamRestorer) flush() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	raw := r.buf.String()
	r.buf.Reset()
	if raw == "" {
		return ""
	}
	r.table.mu.Lock()
	defer r.table.mu.Unlock()
	return r.table.restoreLocked(raw)
}

func splitRestorable(s string) (emit, keep string) {
	if s == "" {
		return "", ""
	}
	if idx := incompletePlaceholderIndex(s); idx >= 0 {
		return s[:idx], s[idx:]
	}
	if idx := trailingPlaceholderPrefix(s); idx >= 0 {
		return s[:idx], s[idx:]
	}
	return s, ""
}

func incompletePlaceholderIndex(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] != '{' {
			continue
		}
		rest := s[i:]
		if strings.HasPrefix(rest, "{{PII_") {
			if loc := placeholderRE.FindStringIndex(rest); loc != nil && loc[0] == 0 {
				continue
			}
			return i
		}
		if prefixOf("{{PII_", rest) {
			return i
		}
	}
	return -1
}

func trailingPlaceholderPrefix(s string) int {
	const maxToken = 48
	if !hasTrailingPlaceholderPrefix(s) {
		return -1
	}
	for i := len(s) - 1; i >= 0 && i >= len(s)-maxToken; i-- {
		if s[i] == '{' {
			return i
		}
	}
	return -1
}

func hasTrailingPlaceholderPrefix(s string) bool {
	prefixes := []string{"{", "{{", "{{P", "{{PI", "{{PII", "{{PII_"}
	for _, p := range prefixes {
		if strings.HasSuffix(s, p) {
			return true
		}
	}
	return false
}

func prefixOf(full, partial string) bool {
	if partial == "" || len(partial) >= len(full) {
		return false
	}
	return strings.HasPrefix(full, partial)
}