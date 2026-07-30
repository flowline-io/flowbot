package agent

import (
	"context"

	agentevent "github.com/flowline-io/flowbot/pkg/agent/event"
	"github.com/flowline-io/flowbot/pkg/agent/loop"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
)

var (
	// ErrMaxSteps is returned when the loop exceeds Config.MaxSteps.
	ErrMaxSteps = loop.ErrMaxSteps
	// ErrAborted is returned when the loop is cancelled.
	ErrAborted = loop.ErrAborted
	// ErrToolNotFound is returned when a tool call names an unknown tool.
	ErrToolNotFound = loop.ErrToolNotFound
	// ErrEmptyContext is returned when Continue is called with no messages.
	ErrEmptyContext = loop.ErrEmptyContext
	// ErrInvalidContinue is returned when Continue cannot resume from the last message.
	ErrInvalidContinue = loop.ErrInvalidContinue
)

type (
	// Agent is a stateful wrapper around the agent loop with queues and subscriptions.
	Agent = loop.Agent
	// Options configures a new Agent instance.
	Options = loop.Options
	// LoopDeps holds runtime dependencies for the agent loop.
	LoopDeps = loop.LoopDeps
	// ModelResolver returns the langchaingo client for a specific model name.
	ModelResolver = loop.ModelResolver
)

// DefaultConfig returns conservative defaults for a new agent run.
func DefaultConfig() Config {
	return loop.DefaultConfig()
}

// NewUserMessage builds a text user message with the current timestamp.
func NewUserMessage(text string) UserMessage {
	return msg.NewUserMessage(text)
}

// NewUserMessageWithParts builds a multimodal user message.
func NewUserMessageWithParts(parts ...ContentPart) UserMessage {
	return msg.NewUserMessageWithParts(parts...)
}

// NewAgent creates an agent with default transforms and optional dependencies.
func NewAgent(opts Options) *Agent {
	return loop.NewAgent(opts)
}

// RunLoop starts a new agent loop from prompt messages.
func RunLoop(ctx context.Context, prompts []AgentMessage, agentCtx *Context, cfg Config, deps LoopDeps, stream *agentevent.Stream) ([]AgentMessage, error) {
	return loop.RunLoop(ctx, prompts, agentCtx, cfg, deps, stream)
}

// RunLoopContinue resumes a loop from existing context without adding prompts.
func RunLoopContinue(ctx context.Context, agentCtx *Context, cfg Config, deps LoopDeps, stream *agentevent.Stream) ([]AgentMessage, error) {
	return loop.RunLoopContinue(ctx, agentCtx, cfg, deps, stream)
}
