package llm

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/bytedance/sonic"
)

// thinkingRequestProfile describes which thinking request fields a model family accepts.
type thinkingRequestProfile struct {
	// SupportsThinkingType injects thinking.type as enabled or disabled.
	SupportsThinkingType bool
	// SupportsReasoningEffort injects DeepSeek-style reasoning_effort.
	SupportsReasoningEffort bool
}

// thinkingRequestProfileFor returns the OpenAI-compatible thinking field profile for a model.
func thinkingRequestProfileFor(modelName string) (thinkingRequestProfile, bool) {
	switch {
	case isDeepSeekV4ReasoningModel(modelName):
		return thinkingRequestProfile{
			SupportsThinkingType:    true,
			SupportsReasoningEffort: true,
		}, true
	case isMiMoReasoningModel(modelName):
		return thinkingRequestProfile{
			SupportsThinkingType: true,
		}, true
	default:
		return thinkingRequestProfile{}, false
	}
}

// thinkingTransport injects OpenAI-compatible thinking parameters into chat
// completion requests according to the model family's thinkingRequestProfile.
type thinkingTransport struct {
	base http.RoundTripper
}

func (t *thinkingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	if req.Body == nil || !strings.Contains(req.URL.Path, "chat/completions") {
		return base.RoundTrip(req)
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	_ = req.Body.Close()

	var payload map[string]any
	if err := sonic.Unmarshal(body, &payload); err != nil {
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.ContentLength = int64(len(body))
		return base.RoundTrip(req)
	}

	applyThinkingRequestFields(payload, ThinkingLevelFromContext(req.Context()))
	injectAssistantReasoningContent(payload, AssistantToolReasoningFromContext(req.Context()))

	patched, err := sonic.Marshal(payload)
	if err != nil {
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.ContentLength = int64(len(body))
		return base.RoundTrip(req)
	}

	req.Body = io.NopCloser(bytes.NewReader(patched))
	req.ContentLength = int64(len(patched))
	return base.RoundTrip(req)
}

// applyThinkingRequestFields mutates a chat-completions payload with provider thinking fields.
func applyThinkingRequestFields(payload map[string]any, level string) {
	modelName, hasModel := payload["model"].(string)
	if !hasModel {
		modelName = ""
	}
	profile, ok := thinkingRequestProfileFor(modelName)
	if !ok {
		// Transport is only attached for known thinking models; treat unknown
		// payloads as thinking.type-only to avoid leaking provider-private fields.
		profile = thinkingRequestProfile{SupportsThinkingType: true}
	}

	enabled := thinkingEnabled(level)
	if profile.SupportsThinkingType {
		if _, hasThinking := payload["thinking"]; !hasThinking {
			if enabled {
				payload["thinking"] = map[string]string{"type": "enabled"}
			} else {
				payload["thinking"] = map[string]string{"type": "disabled"}
			}
		}
	}

	if profile.SupportsReasoningEffort {
		// DeepSeek rejects reasoning_effort=none; omit the field when thinking is off.
		if enabled {
			if _, hasEffort := payload["reasoning_effort"]; !hasEffort {
				payload["reasoning_effort"] = deepSeekReasoningEffort(level)
			}
		} else {
			delete(payload, "reasoning_effort")
		}
		return
	}
	delete(payload, "reasoning_effort")
}

// injectAssistantReasoningContent writes reasoning_content onto prior assistant
// messages that include tool_calls. DeepSeek/MiMo require this field to be echoed
// back after tool results even when the current request disables thinking
// (langchaingo MessageContent has no ReasoningContent field).
func injectAssistantReasoningContent(payload map[string]any, reasoning map[string]string) {
	modelName, hasModel := payload["model"].(string)
	if !hasModel || !needsReasoningContentRoundTrip(modelName) {
		return
	}
	messages, ok := payload["messages"].([]any)
	if !ok {
		return
	}
	for _, raw := range messages {
		msg, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		role, ok := msg["role"].(string)
		if !ok || role != "assistant" || !messageHasToolCalls(msg) {
			continue
		}
		if existing, ok := msg["reasoning_content"].(string); ok && strings.TrimSpace(existing) != "" {
			continue
		}
		msg["reasoning_content"] = reasoningForToolCalls(msg, reasoning)
	}
}

func reasoningForToolCalls(msg map[string]any, reasoning map[string]string) string {
	if len(reasoning) == 0 {
		return ""
	}
	toolCalls, ok := msg["tool_calls"].([]any)
	if !ok {
		return ""
	}
	for _, raw := range toolCalls {
		call, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id, ok := call["id"].(string)
		if !ok || strings.TrimSpace(id) == "" {
			continue
		}
		if text, ok := reasoning[id]; ok {
			return text
		}
	}
	return ""
}

func needsReasoningContentRoundTrip(modelName string) bool {
	return isDeepSeekV4ReasoningModel(modelName) || isMiMoReasoningModel(modelName)
}

func messageHasToolCalls(msg map[string]any) bool {
	raw, ok := msg["tool_calls"]
	if !ok || raw == nil {
		return false
	}
	switch v := raw.(type) {
	case []any:
		return len(v) > 0
	case []map[string]any:
		return len(v) > 0
	default:
		return true
	}
}

// ThinkingHTTPClientForTest exposes the thinking-enabled OpenAI-compatible HTTP client for tests.
func ThinkingHTTPClientForTest(base http.RoundTripper) *http.Client {
	if base == nil {
		base = http.DefaultTransport
	}
	return &http.Client{
		Transport: &toolCallIndexTransport{base: &streamIdleTransport{base: &thinkingTransport{base: base}}},
		Timeout:   llmHTTPTimeout(),
	}
}
