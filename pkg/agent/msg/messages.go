package msg

import (
	"strings"
	"time"
)

// MessageRole identifies the role of an agent message in the session tree.
type MessageRole string

const (
	RoleUser              MessageRole = "user"
	RoleAssistant         MessageRole = "assistant"
	RoleToolResult        MessageRole = "toolResult"
	RoleCustom            MessageRole = "custom"
	RoleBranchSummary     MessageRole = "branchSummary"
	RoleCompactionSummary MessageRole = "compactionSummary"
)

// AgentMessage is the domain message type used throughout the agent loop.
type AgentMessage interface {
	Role() MessageRole
}

// TextPart holds plain text content in a multi-part message.
type TextPart struct {
	Text string
}

// ImagePart holds image content for multimodal messages.
type ImagePart struct {
	MIMEType string
	Data     []byte
	URL      string
}

// ToolCallPart is a tool invocation requested by the assistant.
type ToolCallPart struct {
	ID        string
	Name      string
	Arguments string
}

// ContentPart is a union of message part types.
type ContentPart interface {
	isContentPart()
}

func (TextPart) isContentPart()     {}
func (ImagePart) isContentPart()    {}
func (ToolCallPart) isContentPart() {}

// UserMessage is a user turn, optionally multimodal.
type UserMessage struct {
	Parts     []ContentPart
	Timestamp time.Time
}

// Role returns RoleUser.
func (UserMessage) Role() MessageRole { return RoleUser }

// Usage captures token consumption reported by the LLM provider.
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	CacheRead        int
	CacheWrite       int
}

// AssistantMessage is a model turn with optional text and tool calls.
type AssistantMessage struct {
	Parts      []ContentPart
	Model      string
	StopReason string
	Usage      *Usage
	Timestamp  time.Time
	// TurnDurationMs is elapsed milliseconds for the full agent turn (LLM + tools).
	TurnDurationMs int64
	// ThinkingDurationMs is elapsed milliseconds for the reasoning stream phase.
	ThinkingDurationMs int64
	// ThinkingText is accumulated reasoning output for UI replay after refresh.
	ThinkingText string
	// RunDurationMs is total run milliseconds; set on the final assistant message of a run.
	RunDurationMs int64
}

// Role returns RoleAssistant.
func (AssistantMessage) Role() MessageRole { return RoleAssistant }

// ToolCalls extracts tool call parts from the assistant message.
func (m AssistantMessage) ToolCalls() []ToolCallPart {
	var calls []ToolCallPart
	for _, part := range m.Parts {
		if tc, ok := part.(ToolCallPart); ok {
			calls = append(calls, tc)
		}
	}
	return calls
}

// TextContent concatenates text parts for convenience.
func (m AssistantMessage) TextContent() string {
	var text strings.Builder
	for _, part := range m.Parts {
		if tp, ok := part.(TextPart); ok {
			_, _ = text.WriteString(tp.Text)
		}
	}
	return text.String()
}

// ToolResultMessage carries the result of executing a tool call.
type ToolResultMessage struct {
	ToolCallID string
	Name       string
	Parts      []ContentPart
	IsError    bool
	Timestamp  time.Time
	// DurationMs is elapsed milliseconds for tool execution.
	DurationMs int64
}

// Role returns RoleToolResult.
func (ToolResultMessage) Role() MessageRole { return RoleToolResult }

// CustomMessage is an application-specific message rendered before LLM calls.
type CustomMessage struct {
	CustomType         string
	Parts              []ContentPart
	DisplayOnly        bool
	ExcludeFromContext bool
	Timestamp          time.Time
}

// Role returns RoleCustom.
func (CustomMessage) Role() MessageRole { return RoleCustom }

// BranchSummaryMessage injects branch context after tree navigation.
type BranchSummaryMessage struct {
	Summary   string
	FromID    string
	Timestamp time.Time
}

// Role returns RoleBranchSummary.
func (BranchSummaryMessage) Role() MessageRole { return RoleBranchSummary }

// CompactionSummaryMessage injects compacted history context.
type CompactionSummaryMessage struct {
	Summary      string
	TokensBefore int
	Timestamp    time.Time
}

// Role returns RoleCompactionSummary.
func (CompactionSummaryMessage) Role() MessageRole { return RoleCompactionSummary }
