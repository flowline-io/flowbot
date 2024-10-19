package server

import (
	"fmt"
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
	json "github.com/json-iterator/go"
	"github.com/tmc/langchaingo/llms"
)

// updateMessageHistory updates the message history with the assistant's
// response and requested tool calls.
func updateMessageHistory(messageHistory []llms.MessageContent, resp *llms.ContentResponse) []llms.MessageContent {
	respchoice := resp.Choices[0]

	if respchoice.StopReason != "tool_calls" {
		return nil
	}

	assistantResponse := llms.TextParts(llms.ChatMessageTypeAI, respchoice.Content)
	for _, tc := range respchoice.ToolCalls {
		assistantResponse.Parts = append(assistantResponse.Parts, tc)
	}
	return append(messageHistory, assistantResponse)
}

// executeToolCalls executes the tool calls in the response and returns the
// updated message history.
func executeToolCalls(ctx types.Context, llm llms.Model, messageHistory []llms.MessageContent, resp *llms.ContentResponse) ([]llms.MessageContent, error) {
	flog.Info("[LLM] Executing %d tool calls", len(resp.Choices[0].ToolCalls))
	for _, toolCall := range resp.Choices[0].ToolCalls {
		flog.Info("[LLM] Executing tool call: %s with arguments: %s", toolCall.FunctionCall.Name, toolCall.FunctionCall.Arguments)

		var args types.KV
		if err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &args); err != nil {
			return nil, fmt.Errorf("error unmarshalling arguments: %w", err)
		}

		var err error
		var response string
		ctx.ToolRuleId = toolCall.FunctionCall.Name
		for _, handler := range bots.List() {
			response, err = handler.LangChain(ctx, args)
			if err != nil {
				return nil, fmt.Errorf("error executing tool call: %w", err)
			}
			if response != "" {
				break
			}
		}
		if response == "" {
			return nil, fmt.Errorf("unsupported tool: %s", toolCall.FunctionCall.Name)
		}

		callResponse := llms.MessageContent{
			Role: llms.ChatMessageTypeTool,
			Parts: []llms.ContentPart{
				llms.ToolCallResponse{
					ToolCallID: toolCall.ID,
					Name:       toolCall.FunctionCall.Name,
					Content:    response,
				},
			},
		}
		messageHistory = append(messageHistory, callResponse)
	}

	return messageHistory, nil
}
