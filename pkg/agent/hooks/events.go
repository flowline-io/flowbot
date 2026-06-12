package hooks

import "github.com/flowline-io/flowbot/pkg/agent/msg"

// Event name constants for harness and observation hooks.
const (
	EventBeforeAgentStart = "before_agent_start"
	EventContext          = "context"
	EventToolCall         = "tool_call"
	EventToolResult       = "tool_result"
	EventSavePoint        = "save_point"
	EventContextUsage     = "context_usage"
	EventContextCompacted = "context_compacted"
	EventModelUpdate      = "model_update"
	EventToolsUpdate      = "tools_update"
)

// BeforeAgentStartEvent fires before an agent run begins.
type BeforeAgentStartEvent struct {
	Messages     []msg.AgentMessage
	SystemPrompt string
	ModelName    string
}

// BeforeAgentStartResult can mutate prompts or cancel the run.
type BeforeAgentStartResult struct {
	Messages     []msg.AgentMessage
	SystemPrompt *string
	Cancel       bool
}

// ContextEvent fires before each LLM request with the working message list.
type ContextEvent struct {
	Messages []msg.AgentMessage
}

// ContextResult replaces the message list when Messages is non-nil.
type ContextResult struct {
	Messages []msg.AgentMessage
}

// ToolCallEvent fires before a tool executes.
type ToolCallEvent struct {
	Assistant msg.AssistantMessage
	ToolCall  msg.ToolCallPart
	Args      map[string]any
	Context   *msg.Context
}

// ToolCallResult can block tool execution.
type ToolCallResult struct {
	Block  bool
	Reason string
}

// ToolResultEvent fires after a tool executes.
type ToolResultEvent struct {
	Assistant msg.AssistantMessage
	ToolCall  msg.ToolCallPart
	Args      map[string]any
	Result    msg.ToolResultMessage
	Context   *msg.Context
}

// ToolResultResult patches tool output or requests early loop termination.
type ToolResultResult struct {
	Parts     []msg.ContentPart
	IsError   *bool
	Terminate bool
}

// ContextUsageInfo reports estimated context consumption for observation hooks.
type ContextUsageInfo struct {
	Tokens        int
	ContextWindow int
	Percent       float64
}

// ObservationEvent is a read-only harness notification delivered to Observe handlers.
type ObservationEvent struct {
	Type         string
	Messages     []msg.AgentMessage
	SystemPrompt string
	ModelName    string
	ActiveTools  []string
	ContextUsage *ContextUsageInfo
}
