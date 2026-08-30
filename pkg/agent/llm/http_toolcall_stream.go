package llm

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/bytedance/sonic"
)

// toolCallIndexTransport restores SSE tool-call `index` before langchaingo parses the stream.
type toolCallIndexTransport struct {
	base http.RoundTripper
}

func (t *toolCallIndexTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	resp, err := base.RoundTrip(req)
	if err != nil || resp == nil || resp.Body == nil {
		return resp, err
	}
	if resp.StatusCode != http.StatusOK || req.URL == nil || !isChatCompletionsPath(req.URL.Path) {
		return resp, nil
	}
	if !isChatCompletionSSE(resp) {
		return resp, nil
	}
	resp.Body = newIndexedToolCallReader(resp.Body)
	return resp, nil
}

func isChatCompletionSSE(resp *http.Response) bool {
	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	return strings.Contains(ct, "event-stream")
}

type assembledToolCall struct {
	id        string
	typ       string
	name      string
	arguments string
}

type choiceToolState struct {
	calls map[int]*assembledToolCall
	// lastIdx is the last merged tool index on this choice. Argument-only
	// deltas that omit index attach here so they do not all collapse onto 0.
	lastIdx int
}

type indexedToolCallReader struct {
	inner    io.ReadCloser
	scanBuf  bytes.Buffer
	pending  bytes.Buffer
	byChoice map[int]*choiceToolState
	eof      bool
}

func newIndexedToolCallReader(inner io.ReadCloser) io.ReadCloser {
	return &indexedToolCallReader{inner: inner, byChoice: map[int]*choiceToolState{}}
}

func (r *indexedToolCallReader) Read(p []byte) (int, error) {
	for r.pending.Len() == 0 && !r.eof {
		if err := r.pull(); err != nil {
			if err != io.EOF {
				return 0, err
			}
			if leftover := r.scanBuf.String(); leftover != "" {
				r.handleLine(leftover)
				r.scanBuf.Reset()
			}
			r.flushCalls()
			r.eof = true
			break
		}
	}
	if r.pending.Len() == 0 {
		if r.eof {
			return 0, io.EOF
		}
		return 0, nil
	}
	return r.pending.Read(p)
}

func (r *indexedToolCallReader) pull() error {
	buf := make([]byte, 4096)
	n, err := r.inner.Read(buf)
	if n > 0 {
		_, _ = r.scanBuf.Write(buf[:n])
		for {
			data := r.scanBuf.Bytes()
			idx := bytes.IndexByte(data, '\n')
			if idx < 0 {
				break
			}
			line := string(data[:idx+1])
			r.scanBuf.Next(idx + 1)
			r.handleLine(line)
		}
	}
	return err
}

func (r *indexedToolCallReader) Close() error {
	r.flushCalls()
	return r.inner.Close()
}

func (r *indexedToolCallReader) handleLine(line string) {
	trimmed := strings.TrimRight(line, "\r\n")
	if !strings.HasPrefix(trimmed, "data:") {
		r.writeRaw(line)
		return
	}
	data := strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))
	if data == "[DONE]" {
		r.flushCalls()
		r.writeRaw(line)
		return
	}
	var payload map[string]any
	if err := sonic.Unmarshal([]byte(data), &payload); err != nil {
		r.writeRaw(line)
		return
	}
	ingested := r.mergeToolCalls(payload)
	finish := payloadHasFinishReason(payload)
	if finish {
		r.flushCalls()
	}
	if !ingested {
		r.writeRaw(line)
		return
	}
	if finish || payloadHasVisibleDelta(payload) || payload["usage"] != nil {
		r.writeData(payload)
	}
}

func (r *indexedToolCallReader) mergeToolCalls(payload map[string]any) bool {
	choices, ok := payload["choices"].([]any)
	if !ok || len(choices) == 0 {
		return false
	}
	ingested := false
	for _, rawChoice := range choices {
		choice, ok := rawChoice.(map[string]any)
		if !ok {
			continue
		}
		if r.mergeChoiceToolCalls(choice) {
			ingested = true
		}
	}
	return ingested
}

func (r *indexedToolCallReader) mergeChoiceToolCalls(choice map[string]any) bool {
	delta, ok := choice["delta"].(map[string]any)
	if !ok {
		return false
	}
	rawCalls, ok := delta["tool_calls"].([]any)
	if !ok || len(rawCalls) == 0 {
		return false
	}
	state := r.choiceState(choiceIndex(choice))
	for i, raw := range rawCalls {
		call, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		state.applyDelta(state.resolveIndex(call, i), call)
	}
	delete(delta, "tool_calls")
	if len(delta) == 0 {
		delete(choice, "delta")
	}
	return true
}

func (r *indexedToolCallReader) choiceState(choiceIdx int) *choiceToolState {
	state := r.byChoice[choiceIdx]
	if state == nil {
		state = &choiceToolState{calls: map[int]*assembledToolCall{}, lastIdx: -1}
		r.byChoice[choiceIdx] = state
	}
	return state
}

func (s *choiceToolState) resolveIndex(call map[string]any, fallback int) int {
	if idx, ok := jsonObjectIndex(call); ok {
		return idx
	}
	if hasToolIdentity(call) {
		return fallback
	}
	if s.lastIdx >= 0 {
		return s.lastIdx
	}
	return fallback
}

func (s *choiceToolState) applyDelta(idx int, call map[string]any) {
	item := s.calls[idx]
	if item == nil {
		item = &assembledToolCall{}
		s.calls[idx] = item
	}
	s.lastIdx = idx
	if id, ok := call["id"].(string); ok && id != "" {
		item.id = id
	}
	if typ, ok := call["type"].(string); ok && typ != "" {
		item.typ = typ
	}
	fn, ok := call["function"].(map[string]any)
	if !ok {
		return
	}
	if name, ok := fn["name"].(string); ok && name != "" {
		item.name = name
	}
	if args, ok := fn["arguments"].(string); ok {
		item.arguments += args
	}
}

func (r *indexedToolCallReader) flushCalls() {
	if len(r.byChoice) == 0 {
		return
	}
	maxChoice := maxMapKey(r.byChoice)
	for choiceIdx := 0; choiceIdx <= maxChoice; choiceIdx++ {
		state := r.byChoice[choiceIdx]
		if state == nil || len(state.calls) == 0 {
			continue
		}
		maxTool := maxAssembledKey(state.calls)
		for toolIdx := 0; toolIdx <= maxTool; toolIdx++ {
			item := state.calls[toolIdx]
			if item == nil {
				continue
			}
			r.writeData(completeToolCallEvent(choiceIdx, item))
		}
	}
	r.byChoice = map[int]*choiceToolState{}
}

func completeToolCallEvent(choiceIdx int, item *assembledToolCall) map[string]any {
	typ := item.typ
	if typ == "" {
		typ = "function"
	}
	return map[string]any{
		"choices": []any{
			map[string]any{
				"index": choiceIdx,
				"delta": map[string]any{
					"tool_calls": []any{
						map[string]any{
							"id":   item.id,
							"type": typ,
							"function": map[string]any{
								"name":      item.name,
								"arguments": item.arguments,
							},
						},
					},
				},
			},
		},
	}
}

func (r *indexedToolCallReader) writeData(payload map[string]any) {
	raw, err := sonic.Marshal(payload)
	if err != nil {
		return
	}
	_, _ = r.pending.WriteString("data: ")
	_, _ = r.pending.Write(raw)
	_ = r.pending.WriteByte('\n')
}

func (r *indexedToolCallReader) writeRaw(line string) {
	_, _ = r.pending.WriteString(line)
	if line == "" || line[len(line)-1] != '\n' {
		_ = r.pending.WriteByte('\n')
	}
}

func jsonObjectIndex(m map[string]any) (int, bool) {
	raw, ok := m["index"]
	if !ok || raw == nil {
		return 0, false
	}
	v, ok := raw.(float64)
	if !ok {
		return 0, false
	}
	return int(v), true
}

func choiceIndex(choice map[string]any) int {
	idx, ok := jsonObjectIndex(choice)
	if !ok {
		return 0
	}
	return idx
}

func hasToolIdentity(call map[string]any) bool {
	if id, ok := call["id"].(string); ok && id != "" {
		return true
	}
	if typ, ok := call["type"].(string); ok && typ != "" {
		return true
	}
	fn, ok := call["function"].(map[string]any)
	if !ok {
		return false
	}
	name, ok := fn["name"].(string)
	return ok && name != ""
}

func payloadChoices(payload map[string]any) []map[string]any {
	raw, ok := payload["choices"].([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		choice, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, choice)
	}
	return out
}

func payloadHasFinishReason(payload map[string]any) bool {
	for _, choice := range payloadChoices(payload) {
		reason, ok := choice["finish_reason"].(string)
		if ok && reason != "" {
			return true
		}
	}
	return false
}

func payloadHasVisibleDelta(payload map[string]any) bool {
	for _, choice := range payloadChoices(payload) {
		delta, ok := choice["delta"].(map[string]any)
		if !ok {
			continue
		}
		if hasNonEmptyString(delta, "content") || hasNonEmptyString(delta, "reasoning_content") || hasNonEmptyString(delta, "role") {
			return true
		}
	}
	return false
}

func hasNonEmptyString(m map[string]any, key string) bool {
	value, ok := m[key].(string)
	return ok && value != ""
}

func maxMapKey(m map[int]*choiceToolState) int {
	maxIdx := -1
	for idx := range m {
		if idx > maxIdx {
			maxIdx = idx
		}
	}
	return maxIdx
}

func maxAssembledKey(m map[int]*assembledToolCall) int {
	maxIdx := -1
	for idx := range m {
		if idx > maxIdx {
			maxIdx = idx
		}
	}
	return maxIdx
}

// IndexedToolCallReaderForTest exposes the parallel tool-call SSE rewriter for tests.
func IndexedToolCallReaderForTest(inner io.ReadCloser) io.ReadCloser {
	return newIndexedToolCallReader(inner)
}

// ToolCallIndexTransportForTest exposes the SSE gate + rewriter transport for tests.
func ToolCallIndexTransportForTest(base http.RoundTripper) http.RoundTripper {
	return &toolCallIndexTransport{base: base}
}
