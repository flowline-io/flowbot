package msg

import "github.com/tmc/langchaingo/llms"

// Context holds the mutable state passed through the agent loop.
type Context struct {
	SystemPrompt string
	Messages     []AgentMessage
	ModelName    string
}

// ToolExecutionMode controls how multiple tool calls from one assistant turn run.
type ToolExecutionMode string

const (
	ToolExecutionParallel   ToolExecutionMode = "parallel"
	ToolExecutionSequential ToolExecutionMode = "sequential"
)

// QueueMode controls how queued steering/follow-up messages drain.
type QueueMode string

const (
	QueueAll QueueMode = "all"
	QueueOne QueueMode = "one-at-a-time"
)

// TurnContext is passed to turn-boundary hooks after each assistant response.
type TurnContext struct {
	Message     AssistantMessage
	ToolResults []ToolResultMessage
	Context     *Context
	NewMessages []AgentMessage
}

// TurnUpdate replaces runtime state before the next provider request.
type TurnUpdate struct {
	Context   *Context
	ModelName string
}

// BeforeToolContext is passed to before-tool hooks.
type BeforeToolContext struct {
	Assistant AssistantMessage
	ToolCall  ToolCallPart
	Args      map[string]any
	Context   *Context
}

// BeforeToolResult can block a tool call before execution.
type BeforeToolResult struct {
	Block  bool
	Reason string
}

// AfterToolContext is passed to after-tool hooks.
type AfterToolContext struct {
	Assistant AssistantMessage
	ToolCall  ToolCallPart
	Args      map[string]any
	Result    ToolResultMessage
	Context   *Context
}

// AfterToolResult can patch a tool result or request early termination.
type AfterToolResult struct {
	Parts     []ContentPart
	IsError   *bool
	Terminate bool
}

// TransformContextFn prunes or enriches messages before LLM conversion.
type TransformContextFn func(messages []AgentMessage) ([]AgentMessage, error)

// ConvertToLLMFn converts agent messages to langchaingo message content.
type ConvertToLLMFn func(messages []AgentMessage) ([]llms.MessageContent, error)

// PrepareNextTurnFn refreshes state at turn boundaries.
type PrepareNextTurnFn func(ctx TurnContext) (*TurnUpdate, error)

// ShouldStopAfterTurnFn allows callers to end the loop after a completed turn.
type ShouldStopAfterTurnFn func(ctx TurnContext) (bool, error)

// BeforeToolCallFn runs before a tool executes and may block it.
type BeforeToolCallFn func(ctx BeforeToolContext) (*BeforeToolResult, error)

// AfterToolCallFn runs after a tool executes and may patch its result.
type AfterToolCallFn func(ctx AfterToolContext) (*AfterToolResult, error)

// GetMessagesFn drains steering or follow-up queues at loop checkpoints.
type GetMessagesFn func() ([]AgentMessage, error)

// StopReason describes why an assistant turn ended.
type StopReason string

const (
	StopReasonComplete StopReason = "complete"
	StopReasonError    StopReason = "error"
	StopReasonAborted  StopReason = "aborted"
)
