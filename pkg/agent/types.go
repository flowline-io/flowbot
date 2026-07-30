package agent

import "github.com/flowline-io/flowbot/pkg/agent/msg"

type (
	// MessageRole identifies the role of an agent message in the session tree.
	MessageRole = msg.MessageRole
	// AgentMessage is the domain message type used throughout the agent loop.
	AgentMessage = msg.AgentMessage
	// TextPart holds plain text content in a multi-part message.
	TextPart = msg.TextPart
	// MediaKind classifies multimodal input for modality checks and LLM conversion.
	MediaKind = msg.MediaKind
	// MediaPart holds multimodal media content for user messages.
	MediaPart = msg.MediaPart
	// ToolCallPart is a tool invocation requested by the assistant.
	ToolCallPart = msg.ToolCallPart
	// ContentPart is a union of message part types.
	ContentPart = msg.ContentPart
	// UserMessage is a user turn, optionally multimodal.
	UserMessage = msg.UserMessage
	// AssistantMessage is a model turn with optional text and tool calls.
	AssistantMessage = msg.AssistantMessage
	// ToolResultMessage carries the outcome of a tool invocation.
	ToolResultMessage = msg.ToolResultMessage
	// CustomMessage is an application-defined message in the session tree.
	CustomMessage = msg.CustomMessage
	// BranchSummaryMessage summarizes an abandoned session branch.
	BranchSummaryMessage = msg.BranchSummaryMessage
	// CompactionSummaryMessage summarizes compacted conversation history.
	CompactionSummaryMessage = msg.CompactionSummaryMessage
	// Usage captures token consumption reported by the LLM provider.
	Usage = msg.Usage
	// Context is the in-flight agent conversation state for a loop run.
	Context = msg.Context
	// ToolExecutionMode selects parallel or sequential tool batch execution.
	ToolExecutionMode = msg.ToolExecutionMode
	// QueueMode controls how steering and follow-up queues drain.
	QueueMode = msg.QueueMode
	// TurnContext is the input to PrepareNextTurn / ShouldStopAfterTurn hooks.
	TurnContext = msg.TurnContext
	// TurnUpdate is the mutable result of PrepareNextTurn.
	TurnUpdate = msg.TurnUpdate
	// BeforeToolContext is the input to BeforeToolCall.
	BeforeToolContext = msg.BeforeToolContext
	// BeforeToolResult can cancel or rewrite a tool call before execution.
	BeforeToolResult = msg.BeforeToolResult
	// AfterToolContext is the input to AfterToolCall.
	AfterToolContext = msg.AfterToolContext
	// AfterToolResult can rewrite a tool result or terminate the turn.
	AfterToolResult = msg.AfterToolResult
	// TransformContextFn optionally rewrites messages before ConvertToLLM.
	TransformContextFn = msg.TransformContextFn
	// ConvertToLLMFn converts agent messages to provider message content.
	ConvertToLLMFn = msg.ConvertToLLMFn
	// PrepareNextTurnFn runs after tools and may inject follow-up work.
	PrepareNextTurnFn = msg.PrepareNextTurnFn
	// ShouldStopAfterTurnFn decides whether the inner loop should stop.
	ShouldStopAfterTurnFn = msg.ShouldStopAfterTurnFn
	// BeforeToolCallFn runs before each tool execution.
	BeforeToolCallFn = msg.BeforeToolCallFn
	// AfterToolCallFn runs after each tool execution.
	AfterToolCallFn = msg.AfterToolCallFn
	// GetMessagesFn drains steering or follow-up queues.
	GetMessagesFn = msg.GetMessagesFn
	// StopReason is the model stop reason for an assistant turn.
	StopReason = msg.StopReason
	// Config holds runtime options for an agent loop invocation.
	Config = msg.Config
)

const (
	// RoleUser is a user message role.
	RoleUser = msg.RoleUser
	// RoleAssistant is an assistant message role.
	RoleAssistant = msg.RoleAssistant
	// RoleToolResult is a tool result message role.
	RoleToolResult = msg.RoleToolResult
	// RoleCustom is a custom application message role.
	RoleCustom = msg.RoleCustom
	// RoleBranchSummary is a branch summary message role.
	RoleBranchSummary = msg.RoleBranchSummary
	// RoleCompactionSummary is a compaction summary message role.
	RoleCompactionSummary = msg.RoleCompactionSummary
	// ToolExecutionParallel runs tool calls concurrently.
	ToolExecutionParallel = msg.ToolExecutionParallel
	// ToolExecutionSequential runs tool calls one at a time.
	ToolExecutionSequential = msg.ToolExecutionSequential
	// QueueAll drains every queued steering/follow-up message.
	QueueAll = msg.QueueAll
	// QueueOne drains a single queued message per drain.
	QueueOne = msg.QueueOne
	// StopReasonComplete indicates a normal model completion.
	StopReasonComplete = msg.StopReasonComplete
	// StopReasonError indicates the model stopped due to an error.
	StopReasonError = msg.StopReasonError
	// StopReasonAborted indicates the run was cancelled.
	StopReasonAborted = msg.StopReasonAborted
	// MediaKindImage is image input.
	MediaKindImage = msg.MediaKindImage
	// MediaKindAudio is audio input.
	MediaKindAudio = msg.MediaKindAudio
	// MediaKindVideo is video input.
	MediaKindVideo = msg.MediaKindVideo
)
