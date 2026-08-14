package server

import (
	"bufio"
	"fmt"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
)

// RegisterAgentLLMRoutes mounts the OpenAI-compatible LLM proxy for flowbot-agent.
func RegisterAgentLLMRoutes(a *fiber.App) {
	a.Post("/agent/v1/chat/completions", route.Authorize(route.RequireScope(auth.ScopeAgentHeadless, agentLLMChatCompletions)))
}

func agentLLMChatCompletions(c fiber.Ctx) error {
	var req openAIChatRequest
	if err := sonic.Unmarshal(c.Body(), &req); err != nil {
		return agentLLMError(c, fiber.StatusBadRequest, "invalid_request_error", "invalid json body")
	}
	modelName := strings.TrimSpace(config.ChatAgentChatModel())
	if modelName == "" {
		return agentLLMError(c, fiber.StatusServiceUnavailable, "server_error", "chat model is not configured")
	}

	systemPrompt, messages, err := openAIMessagesToLLM(req.Messages)
	if err != nil {
		return agentLLMError(c, fiber.StatusBadRequest, "invalid_request_error", err.Error())
	}
	tools := openAIToolsToLLM(req.Tools)

	baseCtx := c.Context()
	model, resolvedName, err := agentllm.GetOrCreateModel(baseCtx, modelName)
	if err != nil {
		flog.Error(fmt.Errorf("agent llm proxy: get model: %w", err))
		return agentLLMError(c, fiber.StatusBadGateway, "server_error", "model unavailable")
	}

	temp := 0.0
	if req.Temperature != nil {
		temp = *req.Temperature
	}
	opts := agentllm.StreamOptions{
		ModelName:   resolvedName,
		Temperature: temp,
		MaxTokens:   req.MaxTokens,
		Tools:       tools,
	}

	completionID := "chatcmpl-" + types.Id()
	created := time.Now().Unix()

	if req.Stream {
		// Mirror chatagent SSE: return after SetBodyStreamWriter; long LLM work
		// runs in the stream writer. Complete StreamAssistant before writing any
		// SSE bytes — concurrent Flush during generation truncated langchaingo
		// clients (idle timeout while server already finished).
		c.Set("Content-Type", "text/event-stream")
		c.Set("Cache-Control", "no-cache")
		c.Set("Connection", "keep-alive")
		return c.SendStreamWriter(func(w *bufio.Writer) {
			result, err := agentllm.StreamAssistant(baseCtx, model, systemPrompt, messages, opts)
			if err != nil {
				flog.Error(fmt.Errorf("agent llm proxy: stream: %w", err))
				writeAgentLLMStreamError(w, completionID, resolvedName, created)
				return
			}
			writeAgentLLMStreamResult(w, completionID, resolvedName, created, result)
		})
	}

	// Non-stream JSON completions must wait for the model turn before writing
	// the body (same tradeoff as other sync LLM REST handlers).
	result, err := agentllm.StreamAssistant(baseCtx, model, systemPrompt, messages, opts)
	if err != nil {
		flog.Error(fmt.Errorf("agent llm proxy: complete: %w", err))
		return agentLLMError(c, fiber.StatusBadGateway, "server_error", "completion failed")
	}
	resp := openAIResponseFromResult(completionID, resolvedName, created, result)
	c.Set("Content-Type", "application/json")
	return c.JSON(resp)
}

func writeAgentLLMStreamResult(
	w *bufio.Writer,
	completionID, modelName string,
	created int64,
	result agentllm.AssistantResult,
) {
	if text := result.Content; text != "" {
		if !writeAgentLLMStreamChunk(w, openAIStreamChunk(completionID, modelName, created, &openAIChatMsg{
			Content: &openAIContent{Text: text},
		}, nil)) {
			return
		}
	}
	finish := openAIFinishReason(result)
	if len(result.ToolCalls) > 0 {
		delta := &openAIChatMsg{ToolCalls: llmToolCallsToOpenAI(result.ToolCalls)}
		if !writeAgentLLMStreamChunk(w, openAIStreamChunk(completionID, modelName, created, delta, &finish)) {
			return
		}
	} else if !writeAgentLLMStreamChunk(w, openAIStreamChunk(completionID, modelName, created, &openAIChatMsg{Content: &openAIContent{}}, &finish)) {
		return
	}
	writeAgentLLMDONE(w)
}

func writeAgentLLMStreamError(w *bufio.Writer, completionID, modelName string, created int64) {
	stop := "stop"
	_ = writeAgentLLMStreamChunk(w, openAIStreamChunk(completionID, modelName, created, &openAIChatMsg{Content: &openAIContent{}}, &stop))
	writeAgentLLMDONE(w)
}

func writeAgentLLMStreamChunk(w *bufio.Writer, chunk openAIChatResponse) bool {
	raw, err := sonic.Marshal(chunk)
	if err != nil {
		return false
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", raw); err != nil {
		return false
	}
	return w.Flush() == nil
}

func writeAgentLLMDONE(w *bufio.Writer) {
	_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	_ = w.Flush()
}

func agentLLMError(c fiber.Ctx, status int, errType, message string) error {
	return c.Status(status).JSON(openAIErrorBody{
		Error: openAIErrorDetail{Message: message, Type: errType},
	})
}
