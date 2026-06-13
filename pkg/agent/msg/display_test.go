package msg_test

import (
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/stretchr/testify/assert"
)

func TestIsToolCallPayload(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{
			name: "openai function array",
			text: `[{"id":"call_00_Mxu5KeF0lFBcRijJx1fC1049","type":"function","function":{"name":"run_code","arguments":"{\"code\":\"x\"}"}}]`,
			want: true,
		},
		{
			name: "partial stream prefix",
			text: `[{"id":"call_00_Mxu5KeF0lFBcRijJx1fC1049","type":"function","function":{"name":"run_code","arguments":`,
			want: true,
		},
		{
			name: "plain assistant reply",
			text: "这是第 5 次交互",
			want: false,
		},
		{
			name: "markdown reply",
			text: "# Title\n\n- item one",
			want: false,
		},
		{
			name: "tool status line",
			text: "Running tool: run_code...",
			want: false,
		},
		{
			name: "single function object",
			text: `{"id":"call_01","type":"function","function":{"name":"read_file","arguments":"{}"}}`,
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, msg.IsToolCallPayload(tt.text))
		})
	}
}

func TestSummarizeToolCallPayload(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		want        string
		wantSubstr  []string
		wantAbsent  []string
		wantOneLine bool
	}{
		{
			name: "single tool with args",
			text: `[{"id":"call_00","type":"function","function":{"name":"run_code","arguments":"{\"code\":\"x\"}"}}]`,
			want: `run_code({"code":"x"})`,
		},
		{
			name: "single tool empty args object",
			text: `[{"id":"call_00","type":"function","function":{"name":"run_code","arguments":"{}"}}]`,
			want: `run_code({})`,
		},
		{
			name: "object arguments",
			text: `[{"id":"call_00","type":"function","function":{"name":"web_search","arguments":{"query":"广州天气"}}}]`,
			want: `web_search({"query":"广州天气"})`,
		},
		{
			name: "empty string arguments",
			text: `[{"id":"call_00","type":"function","function":{"name":"web_search","arguments":""}}]`,
			want: `web_search(...)`,
		},
		{
			name: "empty quoted arguments with trailing json",
			text: `[{"id":"call_00","type":"function","function":{"name":"web_search","arguments":""}}]`,
			want: `web_search(...)`,
		},
		{
			name: "multiple tools one line",
			text: `[{"id":"call_00","type":"function","function":{"name":"run_code","arguments":"{}"}},{"id":"call_01","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"a.go\"}"}}]`,
			want: `run_code({}), read_file({"path":"a.go"})`,
		},
		{
			name:        "partial stream uses tool format",
			text:        `[{"id":"call_00","type":"function","function":{"name":"web_search","arguments":`,
			wantSubstr:  []string{"web_search("},
			wantAbsent:  []string{`[{"id":"call_00"`},
			wantOneLine: true,
		},
		{
			name: "corrupted concatenated arrays",
			text: `[{"id":"call_00","type":"function","function":{"name":"web_search","arguments":"{}"}}][{"type":"","function":{"name":"","arguments":"\""}}]`,
			want: `web_search({})`,
		},
		{
			name:        "unknown partial payload",
			text:        `[{"id":"call_00","type":"function"`,
			wantSubstr:  []string{"Running tool..."},
			wantOneLine: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := msg.SummarizeToolCallPayload(tt.text)
			if tt.name == "unknown partial payload" {
				got = msg.SanitizeAssistantDisplayText(tt.text)
			}
			if tt.want != "" {
				assert.Equal(t, tt.want, got)
			}
			for _, want := range tt.wantSubstr {
				assert.Contains(t, got, want)
			}
			for _, absent := range tt.wantAbsent {
				assert.NotContains(t, got, absent)
			}
			if tt.wantOneLine {
				assert.NotContains(t, got, "\n")
			}
		})
	}
}

func TestSummarizeToolCallPayloadTruncatesLongLine(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{name: "long arguments", text: `[{"id":"call_00","type":"function","function":{"name":"run_code","arguments":"` + strings.Repeat("x", 400) + `"}}]`},
		{name: "long partial json", text: `[{"id":"call_00","type":"function","function":{"name":"run_code","arguments":"` + strings.Repeat("y", 400)},
		{name: "long partial via sanitize", text: `[{"id":"call_00","type":"function","function":{"name":"run_code","arguments":"` + strings.Repeat("z", 400)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := msg.SanitizeAssistantDisplayText(tt.text)
			assert.LessOrEqual(t, len(got), 320)
			assert.NotContains(t, got, "\n")
			assert.Contains(t, got, "run_code(")
			assert.NotContains(t, got, `[{"id"`)
		})
	}
}

func TestSummarizeToolCallParts(t *testing.T) {
	tests := []struct {
		name  string
		calls []msg.ToolCallPart
		want  string
	}{
		{
			name: "structured args",
			calls: []msg.ToolCallPart{{
				ID:        "call_00",
				Name:      "web_search",
				Arguments: `{"query":"广州天气"}`,
			}},
			want: `web_search({"query":"广州天气"})`,
		},
		{
			name: "multiple calls",
			calls: []msg.ToolCallPart{
				{ID: "call_00", Name: "run_code", Arguments: `{"code":"x"}`},
				{ID: "call_01", Name: "read_file", Arguments: `{"path":"a.go"}`},
			},
			want: `run_code({"code":"x"}), read_file({"path":"a.go"})`,
		},
		{
			name:  "missing args placeholder",
			calls: []msg.ToolCallPart{{ID: "call_00", Name: "web_search"}},
			want:  `web_search(...)`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, msg.SummarizeToolCallParts(tt.calls))
		})
	}
}

func TestAssistantDisplayTextPrefersToolCalls(t *testing.T) {
	tests := []struct {
		name string
		msg  msg.AssistantMessage
		want string
	}{
		{
			name: "tool parts over empty json text",
			msg: msg.AssistantMessage{
				Parts: []msg.ContentPart{
					msg.TextPart{Text: `[{"id":"call_00","type":"function","function":{"name":"web_search","arguments":""}}]`},
					msg.ToolCallPart{ID: "call_00", Name: "web_search", Arguments: `{"query":"广州天气"}`},
				},
			},
			want: `web_search({"query":"广州天气"})`,
		},
		{
			name: "plain text reply",
			msg: msg.AssistantMessage{
				Parts: []msg.ContentPart{msg.TextPart{Text: "hello"}},
			},
			want: "hello",
		},
		{
			name: "json object arguments",
			msg: msg.AssistantMessage{
				Parts: []msg.ContentPart{
					msg.TextPart{Text: `[{"id":"call_00","type":"function","function":{"name":"web_search","arguments":{"query":"广州天气"}}}]`},
				},
			},
			want: `web_search({"query":"广州天气"})`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, msg.AssistantDisplayText(tt.msg))
		})
	}
}

func TestSanitizeAssistantDisplayText(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "summarizes tool payload",
			text: `[{"id":"call_00","type":"function","function":{"name":"run_code","arguments":"{}"}}]`,
			want: `run_code({})`,
		},
		{
			name: "keeps natural language",
			text: "hello world",
			want: "hello world",
		},
		{
			name: "preserves surrounding whitespace for normal text",
			text: "  answer text  ",
			want: "  answer text  ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, msg.SanitizeAssistantDisplayText(tt.text))
		})
	}
}
