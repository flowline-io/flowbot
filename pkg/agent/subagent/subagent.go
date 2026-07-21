// Package subagent runs a specialized agent loop in isolation so a primary agent
// can delegate a self-contained task and receive only the final result back.
package subagent

import (
	"context"
	"errors"
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"go.opentelemetry.io/otel/attribute"

	"github.com/flowline-io/flowbot/pkg/agent"
	agentevent "github.com/flowline-io/flowbot/pkg/agent/event"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	fbtrace "github.com/flowline-io/flowbot/pkg/trace"
)

// ErrMaxDepth is returned when subagent nesting exceeds the configured delegation depth.
var ErrMaxDepth = errors.New("subagent: max delegation depth exceeded")

// ErrNoModel is returned when a subagent run is requested without an LLM model.
var ErrNoModel = errors.New("subagent: model is required")

// Definition describes a specialized subagent that the primary agent can delegate to.
// Tools is an allowlist of tool names exposed to the subagent; empty means all
// tools registered on the provided registry.
type Definition struct {
	// Name is the unique subagent identifier referenced by the task tool.
	Name string
	// Description tells the primary agent when to delegate to this subagent.
	Description string
	// SystemPrompt is the isolated instruction set for the subagent loop.
	SystemPrompt string
	// Tools restricts which registered tools the subagent may call.
	Tools []string
	// Skills restricts which agent skills are injected into the subagent prompt and read_skill allowlist.
	Skills []string
	// Model optionally overrides the model used for the subagent loop.
	Model string
}

// Deps carries the runtime dependencies required to run a subagent loop in isolation.
type Deps struct {
	// Model is the resolved LLM client for the subagent loop.
	Model llms.Model
	// Registry holds the tools available to the subagent after allowlist filtering.
	Registry *tool.Registry
	// Config is the loop configuration; defaults are applied when zero.
	Config agent.Config
	// Depth is the current delegation depth of the caller (0 for the primary agent).
	Depth int
	// MaxDepth caps nested delegation; values <= 0 default to 1.
	MaxDepth int
}

// Result is the outcome of a completed subagent run.
type Result struct {
	// Text is the concatenated assistant text produced by the subagent.
	Text string
	// Messages are all messages generated during the subagent run.
	Messages []agent.AgentMessage
}

// ProgressFn receives human-readable progress updates from the subagent run.
type ProgressFn func(update string)

// Run executes a subagent in a fresh, isolated context and returns its final
// assistant text. The run does not share the parent conversation and is not
// persisted to any session tree, matching the mainstream subagent contract.
func Run(ctx context.Context, def Definition, deps Deps, prompt string, onProgress ProgressFn) (Result, error) {
	ctx, span := fbtrace.StartSpan(ctx, "agent.subagent",
		attribute.String("agent.subagent.name", def.Name),
	)
	defer span.End()

	maxDepth := deps.MaxDepth
	if maxDepth <= 0 {
		maxDepth = 1
	}
	if deps.Depth >= maxDepth {
		return Result{}, ErrMaxDepth
	}
	if deps.Model == nil {
		return Result{}, ErrNoModel
	}

	agentCtx := &agent.Context{
		SystemPrompt: def.SystemPrompt,
		ModelName:    def.Model,
	}
	prompts := []agent.AgentMessage{agent.NewUserMessage(prompt)}

	stream := agentevent.NewStream(64)
	done := make(chan struct{})
	var runMessages []agent.AgentMessage
	var runErr error
	go func() {
		runMessages, runErr = agent.RunLoop(ctx, prompts, agentCtx, deps.Config, agent.LoopDeps{
			Model:    deps.Model,
			Registry: deps.Registry,
		}, stream)
		stream.End(nil, runErr)
		close(done)
	}()

	for ev := range stream.Events() {
		forwardProgress(ev, onProgress)
	}
	<-done

	result := Result{Messages: runMessages, Text: FinalText(runMessages)}
	if runErr != nil {
		return result, runErr
	}
	return result, nil
}

// FinalText returns the concatenated text of the last assistant message in the run.
func FinalText(messages []agent.AgentMessage) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if assistant, ok := messages[i].(agent.AssistantMessage); ok {
			if text := assistant.TextContent(); text != "" {
				return text
			}
		}
	}
	return ""
}

func forwardProgress(ev agentevent.Event, onProgress ProgressFn) {
	if onProgress == nil {
		return
	}
	switch ev.Type {
	case agentevent.TypeToolExecutionStart:
		if name := toolCallName(ev.ToolCall); name != "" {
			onProgress(fmt.Sprintf("running tool: %s", name))
		}
	case agentevent.TypeToolExecutionUpdate:
		if ev.Update != "" {
			onProgress(ev.Update)
		}
	}
}

func toolCallName(toolCall any) string {
	if call, ok := toolCall.(msg.ToolCallPart); ok {
		return call.Name
	}
	return ""
}
