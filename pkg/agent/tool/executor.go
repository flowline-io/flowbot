package tool

import (
	"context"
	"fmt"
	"sync"

	"github.com/bytedance/sonic"
	agentevent "github.com/flowline-io/flowbot/pkg/agent/event"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
)

// BatchRequest describes one tool execution batch from an assistant turn.
type BatchRequest struct {
	Assistant msg.AssistantMessage
	Context   *msg.Context
	Registry  *Registry
	Mode      msg.ToolExecutionMode
	Before    msg.BeforeToolCallFn
	After     msg.AfterToolCallFn
	Emit      func(context.Context, agentevent.Event) error
}

// BatchResult is the outcome of executing one assistant tool batch.
type BatchResult struct {
	Messages  []msg.ToolResultMessage
	Terminate bool
}

// ExecuteBatch runs tool calls from an assistant message and returns tool result messages.
func ExecuteBatch(ctx context.Context, req BatchRequest) (BatchResult, error) {
	calls := req.Assistant.ToolCalls()
	if len(calls) == 0 {
		return BatchResult{}, nil
	}

	if req.Mode == msg.ToolExecutionSequential {
		return executeSequential(ctx, req, calls)
	}
	return executeParallel(ctx, req, calls)
}

func executeSequential(ctx context.Context, req BatchRequest, calls []msg.ToolCallPart) (BatchResult, error) {
	result := BatchResult{}
	for _, call := range calls {
		toolResult, terminate, err := executeOne(ctx, req, call)
		if err != nil {
			return result, err
		}
		result.Messages = append(result.Messages, toolResult)
		if terminate {
			result.Terminate = true
		}
	}
	return result, nil
}

func executeParallel(ctx context.Context, req BatchRequest, calls []msg.ToolCallPart) (BatchResult, error) {
	prepared := make([]preparedCall, 0, len(calls))
	for _, call := range calls {
		item, err := prepareCall(ctx, req, call)
		if err != nil {
			return BatchResult{}, err
		}
		prepared = append(prepared, item)
	}

	type execResult struct {
		index     int
		result    msg.ToolResultMessage
		terminate bool
		err       error
	}

	results := make([]execResult, len(prepared))
	var wg sync.WaitGroup
	wg.Add(len(prepared))

	for i, item := range prepared {
		go func(idx int, preparedItem preparedCall) {
			defer wg.Done()
			toolResult, terminate, err := runPrepared(ctx, req, preparedItem)
			results[idx] = execResult{index: idx, result: toolResult, terminate: terminate, err: err}
		}(i, item)
	}
	wg.Wait()

	batch := BatchResult{}
	for _, item := range results {
		if item.err != nil {
			return batch, item.err
		}
		batch.Messages = append(batch.Messages, item.result)
		if item.terminate {
			batch.Terminate = true
		}
	}
	return batch, nil
}

type preparedCall struct {
	call    msg.ToolCallPart
	args    map[string]any
	tool    Tool
	blocked bool
	reason  string
}

func prepareCall(_ context.Context, req BatchRequest, call msg.ToolCallPart) (preparedCall, error) {
	args, err := parseArgs(call.Arguments)
	if err != nil {
		return preparedCall{}, fmt.Errorf("tool executor: parse args for %q: %w", call.Name, err)
	}

	t, ok := req.Registry.Get(call.Name)
	if !ok {
		return preparedCall{
			call: call,
			args: args,
		}, nil
	}

	if req.Before != nil {
		before, err := req.Before(msg.BeforeToolContext{
			Assistant: req.Assistant,
			ToolCall:  call,
			Args:      args,
			Context:   req.Context,
		})
		if err != nil {
			return preparedCall{}, err
		}
		if before != nil && before.Block {
			return preparedCall{
				call:    call,
				args:    args,
				tool:    t,
				blocked: true,
				reason:  before.Reason,
			}, nil
		}
	}

	return preparedCall{call: call, args: args, tool: t}, nil
}

func executeOne(ctx context.Context, req BatchRequest, call msg.ToolCallPart) (msg.ToolResultMessage, bool, error) {
	prepared, err := prepareCall(ctx, req, call)
	if err != nil {
		return msg.ToolResultMessage{}, false, err
	}
	return runPrepared(ctx, req, prepared)
}

func runPrepared(ctx context.Context, req BatchRequest, prepared preparedCall) (msg.ToolResultMessage, bool, error) {
	call := prepared.call
	if prepared.blocked {
		reason := prepared.reason
		if reason == "" {
			reason = "tool call blocked"
		}
		return blockedResult(call, reason), false, nil
	}

	t, ok := req.Registry.Get(call.Name)
	if !ok {
		return errorResult(call, fmt.Sprintf("%s: %q", msg.ErrToolNotFound.Error(), call.Name)), false, nil
	}
	prepared.tool = t

	if req.Emit != nil {
		_ = req.Emit(ctx, agentevent.Event{Type: agentevent.TypeToolExecutionStart, ToolCall: call})
	}

	toolResult, err := t.Execute(ctx, call.ID, prepared.args, func(update string) error {
		if req.Emit == nil {
			return nil
		}
		return req.Emit(ctx, agentevent.Event{
			Type:     agentevent.TypeToolExecutionUpdate,
			ToolCall: call,
			Update:   update,
		})
	})
	if err != nil {
		toolResult = errorResult(call, err.Error())
	}
	toolResult.ToolCallID = call.ID
	toolResult.Name = call.Name

	terminate := false
	if req.After != nil {
		after, afterErr := req.After(msg.AfterToolContext{
			Assistant: req.Assistant,
			ToolCall:  call,
			Args:      prepared.args,
			Result:    toolResult,
			Context:   req.Context,
		})
		if afterErr != nil {
			return msg.ToolResultMessage{}, false, afterErr
		}
		if after != nil {
			if after.Parts != nil {
				toolResult.Parts = after.Parts
			}
			if after.IsError != nil {
				toolResult.IsError = *after.IsError
			}
			terminate = after.Terminate
		}
	}

	if req.Emit != nil {
		_ = req.Emit(ctx, agentevent.Event{
			Type:       agentevent.TypeToolExecutionEnd,
			ToolCall:   call,
			ToolResult: toolResult,
		})
	}

	return toolResult, terminate, nil
}

func parseArgs(raw string) (map[string]any, error) {
	if raw == "" {
		return map[string]any{}, nil
	}
	var args map[string]any
	if err := sonic.Unmarshal([]byte(raw), &args); err != nil {
		return nil, err
	}
	return args, nil
}

func blockedResult(call msg.ToolCallPart, reason string) msg.ToolResultMessage {
	return msg.ToolResultMessage{
		ToolCallID: call.ID,
		Name:       call.Name,
		Parts:      []msg.ContentPart{msg.TextPart{Text: reason}},
		IsError:    true,
	}
}

func errorResult(call msg.ToolCallPart, message string) msg.ToolResultMessage {
	return msg.ToolResultMessage{
		ToolCallID: call.ID,
		Name:       call.Name,
		Parts:      []msg.ContentPart{msg.TextPart{Text: message}},
		IsError:    true,
	}
}
