package server

import (
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/tmc/langchaingo/llms"

	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
)

// OpenAI-compatible chat completions DTOs for /agent/v1/chat/completions.

type openAIChatRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIChatMsg `json:"messages"`
	Tools       []openAIToolDef `json:"tools,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	Temperature *float64        `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
}

type openAIChatMsg struct {
	Role       string           `json:"role"`
	Content    *openAIContent   `json:"content"`
	Name       string           `json:"name,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
}

type openAIContent struct {
	Text  string
	Parts []openAIContentPart
}

type openAIContentPart struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

func (c *openAIContent) UnmarshalJSON(data []byte) error {
	c.Text = ""
	c.Parts = nil
	if string(data) == "null" {
		return nil
	}
	var s string
	if err := sonic.Unmarshal(data, &s); err == nil {
		c.Text = s
		return nil
	}
	var parts []openAIContentPart
	if err := sonic.Unmarshal(data, &parts); err != nil {
		return err
	}
	c.Parts = parts
	var b strings.Builder
	for _, p := range parts {
		if p.Type == "text" || p.Type == "" {
			_, _ = b.WriteString(p.Text)
		}
	}
	c.Text = b.String()
	return nil
}

func (c *openAIContent) MarshalJSON() ([]byte, error) {
	if c == nil {
		return []byte("null"), nil
	}
	return sonic.Marshal(c.Text)
}

type openAIToolDef struct {
	Type     string             `json:"type"`
	Function *openAIFunctionDef `json:"function,omitempty"`
}

type openAIFunctionDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type openAIToolCall struct {
	ID       string              `json:"id"`
	Type     string              `json:"type"`
	Index    *int                `json:"index,omitempty"`
	Function *openAIFunctionCall `json:"function,omitempty"`
}

type openAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openAIChatResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []openAIChatChoice `json:"choices"`
	Usage   *openAIUsage       `json:"usage,omitempty"`
}

type openAIChatChoice struct {
	Index        int            `json:"index"`
	Message      *openAIChatMsg `json:"message,omitempty"`
	Delta        *openAIChatMsg `json:"delta,omitempty"`
	FinishReason *string        `json:"finish_reason"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type openAIErrorBody struct {
	Error openAIErrorDetail `json:"error"`
}

type openAIErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

func (c *openAIContent) string() string {
	if c == nil {
		return ""
	}
	return c.Text
}

func openAIMessagesToLLM(messages []openAIChatMsg) (systemPrompt string, out []llms.MessageContent, err error) {
	out = make([]llms.MessageContent, 0, len(messages))
	for i, m := range messages {
		role := strings.ToLower(strings.TrimSpace(m.Role))
		switch role {
		case "system":
			text := strings.TrimSpace(m.Content.string())
			if systemPrompt == "" {
				systemPrompt = text
			} else if text != "" {
				systemPrompt = systemPrompt + "\n\n" + text
			}
		case "user":
			text := m.Content.string()
			out = append(out, llms.TextParts(llms.ChatMessageTypeHuman, text))
		case "assistant":
			parts := make([]llms.ContentPart, 0, 1+len(m.ToolCalls))
			if text := m.Content.string(); text != "" {
				parts = append(parts, llms.TextContent{Text: text})
			}
			for _, tc := range m.ToolCalls {
				name, args := "", ""
				if tc.Function != nil {
					name = tc.Function.Name
					args = tc.Function.Arguments
				}
				parts = append(parts, llms.ToolCall{
					ID:   tc.ID,
					Type: "function",
					FunctionCall: &llms.FunctionCall{
						Name:      name,
						Arguments: args,
					},
				})
			}
			if len(parts) == 0 {
				parts = append(parts, llms.TextContent{Text: ""})
			}
			out = append(out, llms.MessageContent{Role: llms.ChatMessageTypeAI, Parts: parts})
		case "tool":
			name := m.Name
			out = append(out, llms.MessageContent{
				Role: llms.ChatMessageTypeTool,
				Parts: []llms.ContentPart{
					llms.ToolCallResponse{
						ToolCallID: m.ToolCallID,
						Name:       name,
						Content:    m.Content.string(),
					},
				},
			})
		default:
			return "", nil, fmt.Errorf("unsupported message role %q at index %d", m.Role, i)
		}
	}
	return systemPrompt, out, nil
}

func openAIToolsToLLM(tools []openAIToolDef) []llms.Tool {
	if len(tools) == 0 {
		return nil
	}
	out := make([]llms.Tool, 0, len(tools))
	for _, t := range tools {
		if t.Function == nil {
			continue
		}
		out = append(out, llms.Tool{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  t.Function.Parameters,
			},
		})
	}
	return out
}

func openAIResponseFromResult(id, modelName string, created int64, result agentllm.AssistantResult) openAIChatResponse {
	finish := openAIFinishReason(result)
	msg := &openAIChatMsg{
		Role:    "assistant",
		Content: &openAIContent{Text: result.Content},
	}
	if len(result.ToolCalls) > 0 {
		msg.ToolCalls = llmToolCallsToOpenAI(result.ToolCalls)
	}
	resp := openAIChatResponse{
		ID:      id,
		Object:  "chat.completion",
		Created: created,
		Model:   modelName,
		Choices: []openAIChatChoice{{
			Index:        0,
			Message:      msg,
			FinishReason: &finish,
		}},
	}
	if result.Usage != nil {
		resp.Usage = &openAIUsage{
			PromptTokens:     result.Usage.PromptTokens,
			CompletionTokens: result.Usage.CompletionTokens,
			TotalTokens:      result.Usage.TotalTokens,
		}
	}
	return resp
}

func openAIFinishReason(result agentllm.AssistantResult) string {
	if result.StopReason == "tool_calls" || len(result.ToolCalls) > 0 {
		return "tool_calls"
	}
	return "stop"
}

func llmToolCallsToOpenAI(calls []llms.ToolCall) []openAIToolCall {
	out := make([]openAIToolCall, 0, len(calls))
	for i, tc := range calls {
		idx := i
		name, args := "", ""
		if tc.FunctionCall != nil {
			name = tc.FunctionCall.Name
			args = tc.FunctionCall.Arguments
		}
		typ := tc.Type
		if typ == "" {
			typ = "function"
		}
		out = append(out, openAIToolCall{
			ID:    tc.ID,
			Type:  typ,
			Index: &idx,
			Function: &openAIFunctionCall{
				Name:      name,
				Arguments: args,
			},
		})
	}
	return out
}

func openAIStreamChunk(id, modelName string, created int64, delta *openAIChatMsg, finish *string) openAIChatResponse {
	return openAIChatResponse{
		ID:      id,
		Object:  "chat.completion.chunk",
		Created: created,
		Model:   modelName,
		Choices: []openAIChatChoice{{
			Index:        0,
			Delta:        delta,
			FinishReason: finish,
		}},
	}
}
