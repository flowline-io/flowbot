package llm_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

func TestIndexedToolCallReaderRewritesStream(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		sse            string
		wantIDs        []string
		wantArgs       []string
		wantContains   []string
		wantNotContain string
	}{
		{
			name:         "splits two indexed parallel argument deltas",
			sse:          indexedParallelSSE(2),
			wantIDs:      []string{"call_00", "call_01"},
			wantArgs:     []string{`{"path":"f0.go"}`, `{"path":"f1.go"}`},
			wantContains: []string{`"reasoning_content":"plan"`, `finish_reason`, `[DONE]`},
		},
		{
			name:     "keeps twelve indexed parallel arguments distinct",
			sse:      indexedParallelSSE(12),
			wantIDs:  indexedCallIDs(12),
			wantArgs: indexedCallArgs(12),
		},
		{
			name: "argument deltas without index attach to last merged tool",
			sse: strings.Join([]string{
				`data: {"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_a","type":"function","function":{"name":"read_file","arguments":""}}]}}]}`,
				`data: {"choices":[{"index":0,"delta":{"tool_calls":[{"index":1,"id":"call_b","type":"function","function":{"name":"list_dir","arguments":""}}]}}]}`,
				`data: {"choices":[{"index":0,"delta":{"tool_calls":[{"function":{"arguments":"{\"path\":\".\"}"}}]}}]}`,
				`data: {"choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`,
				`data: [DONE]`,
				"",
			}, "\n"),
			wantIDs:        []string{"call_a", "call_b"},
			wantArgs:       []string{"", `{"path":"."}`},
			wantNotContain: `{"path":"."}{"path":`,
		},
		{
			name: "rewrites tool calls on non-zero choice index",
			sse: strings.Join([]string{
				`data: {"choices":[{"index":1,"delta":{"tool_calls":[{"index":0,"id":"call_z","type":"function","function":{"name":"echo","arguments":"{}"}}]},"finish_reason":"tool_calls"}]}`,
				`data: [DONE]`,
				"",
			}, "\n"),
			wantIDs:  []string{"call_z"},
			wantArgs: []string{`{}`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out, err := io.ReadAll(llm.IndexedToolCallReaderForTest(io.NopCloser(strings.NewReader(tt.sse))))
			require.NoError(t, err)
			text := string(out)
			got := sseToolCalls(t, text)
			require.Len(t, got, len(tt.wantIDs))
			for i, call := range got {
				assert.Equal(t, tt.wantIDs[i], call.id)
				if tt.wantArgs[i] == "" {
					assert.Empty(t, call.args)
					continue
				}
				assert.JSONEq(t, tt.wantArgs[i], call.args)
			}
			for _, snippet := range tt.wantContains {
				assert.Contains(t, text, snippet)
			}
			if tt.wantNotContain != "" {
				assert.NotContains(t, text, tt.wantNotContain)
			}
		})
	}
}

func TestLangchaingoAssemblesIndexedParallelToolCalls(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		count    int
		wantIDs  []string
		wantArgs []string
	}{
		{
			name:     "two indexed parallel calls",
			count:    2,
			wantIDs:  indexedCallIDs(2),
			wantArgs: indexedCallArgs(2),
		},
		{
			name:     "twelve indexed parallel calls",
			count:    12,
			wantIDs:  indexedCallIDs(12),
			wantArgs: indexedCallArgs(12),
		},
		{
			name:     "three indexed parallel calls",
			count:    3,
			wantIDs:  indexedCallIDs(3),
			wantArgs: indexedCallArgs(3),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			model, err := openai.New(
				openai.WithToken("test"),
				openai.WithModel("mimo-v2.5"),
				openai.WithBaseURL("https://api.example.com/v1"),
				openai.WithHTTPClient(llm.ThinkingHTTPClientForTest(&scriptedStreamTripper{body: indexedParallelSSE(tt.count)})),
			)
			require.NoError(t, err)

			resp, err := model.GenerateContent(context.Background(), []llms.MessageContent{
				llms.TextParts(llms.ChatMessageTypeHuman, "hi"),
			}, llms.WithModel("mimo-v2.5"), llms.WithStreamingFunc(func(context.Context, []byte) error {
				return nil
			}))
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.NotEmpty(t, resp.Choices)
			calls := resp.Choices[0].ToolCalls
			require.Len(t, calls, tt.count)
			for i, call := range calls {
				require.NotNil(t, call.FunctionCall)
				assert.Equal(t, tt.wantIDs[i], call.ID)
				assert.JSONEq(t, tt.wantArgs[i], call.FunctionCall.Arguments)
			}
		})
	}
}

func TestToolCallIndexTransportLeavesNonSSEUnchanged(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		contentType string
		body        string
	}{
		{
			name:        "json completion",
			contentType: "application/json",
			body:        `{"id":"x","choices":[{"message":{"role":"assistant","content":"ok"}}]}`,
		},
		{
			name:        "json without trailing newline",
			contentType: "application/json; charset=utf-8",
			body:        `{"id":"y","object":"chat.completion"}`,
		},
		{
			name:        "empty content type is not rewritten",
			contentType: "",
			body:        `{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_a","type":"function","function":{"name":"echo","arguments":"{}"}}]}}]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			transport := llm.ToolCallIndexTransportForTest(&staticBodyTripper{
				status:      http.StatusOK,
				contentType: tt.contentType,
				body:        tt.body,
			})
			req, err := http.NewRequest(http.MethodPost, "https://api.example.com/v1/chat/completions", http.NoBody)
			require.NoError(t, err)
			resp, err := transport.RoundTrip(req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			got, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, tt.body, string(got))
		})
	}
}

type sseToolCall struct {
	id   string
	args string
}

func sseToolCalls(t *testing.T, body string) []sseToolCall {
	t.Helper()
	var out []sseToolCall
	for _, line := range strings.Split(body, "\n") {
		out = append(out, sseToolCallsInLine(line)...)
	}
	return out
}

func sseToolCallsInLine(line string) []sseToolCall {
	data := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "data:"))
	if data == "" || data == "[DONE]" {
		return nil
	}
	var payload map[string]any
	if err := sonic.Unmarshal([]byte(data), &payload); err != nil {
		return nil
	}
	choices, ok := payload["choices"].([]any)
	if !ok {
		return nil
	}
	var out []sseToolCall
	for _, rawChoice := range choices {
		out = append(out, sseToolCallsInChoice(rawChoice)...)
	}
	return out
}

func sseToolCallsInChoice(rawChoice any) []sseToolCall {
	choice, ok := rawChoice.(map[string]any)
	if !ok {
		return nil
	}
	delta, ok := choice["delta"].(map[string]any)
	if !ok {
		return nil
	}
	rawCalls, ok := delta["tool_calls"].([]any)
	if !ok {
		return nil
	}
	var out []sseToolCall
	for _, rawCall := range rawCalls {
		call, ok := sseToolCallFromRaw(rawCall)
		if !ok {
			continue
		}
		out = append(out, call)
	}
	return out
}

func sseToolCallFromRaw(rawCall any) (sseToolCall, bool) {
	call, ok := rawCall.(map[string]any)
	if !ok {
		return sseToolCall{}, false
	}
	fn, ok := call["function"].(map[string]any)
	if !ok {
		return sseToolCall{}, false
	}
	id, ok := call["id"].(string)
	if !ok {
		id = ""
	}
	args, ok := fn["arguments"].(string)
	if !ok {
		args = ""
	}
	if id == "" && args == "" {
		return sseToolCall{}, false
	}
	return sseToolCall{id: id, args: args}, true
}

func indexedParallelSSE(n int) string {
	var b strings.Builder
	_, _ = b.WriteString(`data: {"choices":[{"index":0,"delta":{"role":"assistant","reasoning_content":"plan"}}]}` + "\n")
	for i := 0; i < n; i++ {
		_, _ = fmt.Fprintf(&b, `data: {"choices":[{"index":0,"delta":{"tool_calls":[{"index":%d,"id":%q,"type":"function","function":{"name":"read_file","arguments":""}}]}}]}`+"\n", i, indexedCallID(i))
	}
	for i := 0; i < n; i++ {
		args := strings.ReplaceAll(indexedCallArg(i), `"`, `\"`)
		_, _ = fmt.Fprintf(&b, `data: {"choices":[{"index":0,"delta":{"tool_calls":[{"index":%d,"function":{"arguments":"%s"}}]}}]}`+"\n", i, args)
	}
	_, _ = b.WriteString(`data: {"choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}` + "\n")
	_, _ = b.WriteString("data: [DONE]\n")
	return b.String()
}

func indexedCallIDs(n int) []string {
	out := make([]string, n)
	for i := 0; i < n; i++ {
		out[i] = indexedCallID(i)
	}
	return out
}

func indexedCallArgs(n int) []string {
	out := make([]string, n)
	for i := 0; i < n; i++ {
		out[i] = indexedCallArg(i)
	}
	return out
}

func indexedCallID(i int) string {
	return fmt.Sprintf("call_%02d", i)
}

func indexedCallArg(i int) string {
	return fmt.Sprintf(`{"path":"f%d.go"}`, i)
}

type scriptedStreamTripper struct {
	body string
}

func (s *scriptedStreamTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(s.body)),
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Request:    req,
	}, nil
}

type staticBodyTripper struct {
	status      int
	contentType string
	body        string
}

func (s *staticBodyTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	header := make(http.Header)
	if s.contentType != "" {
		header.Set("Content-Type", s.contentType)
	}
	return &http.Response{
		StatusCode: s.status,
		Body:       io.NopCloser(strings.NewReader(s.body)),
		Header:     header,
		Request:    req,
	}, nil
}
