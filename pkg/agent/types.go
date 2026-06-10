package agent

import "github.com/flowline-io/flowbot/pkg/agent/msg"

type (
	MessageRole              = msg.MessageRole
	AgentMessage             = msg.AgentMessage
	TextPart                 = msg.TextPart
	ImagePart                = msg.ImagePart
	ToolCallPart             = msg.ToolCallPart
	ContentPart              = msg.ContentPart
	UserMessage              = msg.UserMessage
	AssistantMessage         = msg.AssistantMessage
	ToolResultMessage        = msg.ToolResultMessage
	CustomMessage            = msg.CustomMessage
	BranchSummaryMessage     = msg.BranchSummaryMessage
	CompactionSummaryMessage = msg.CompactionSummaryMessage
	Context                  = msg.Context
	ToolExecutionMode        = msg.ToolExecutionMode
	QueueMode                = msg.QueueMode
	TurnContext              = msg.TurnContext
	TurnUpdate               = msg.TurnUpdate
	BeforeToolContext        = msg.BeforeToolContext
	BeforeToolResult         = msg.BeforeToolResult
	AfterToolContext         = msg.AfterToolContext
	AfterToolResult          = msg.AfterToolResult
	TransformContextFn       = msg.TransformContextFn
	ConvertToLLMFn           = msg.ConvertToLLMFn
	PrepareNextTurnFn        = msg.PrepareNextTurnFn
	ShouldStopAfterTurnFn    = msg.ShouldStopAfterTurnFn
	BeforeToolCallFn         = msg.BeforeToolCallFn
	AfterToolCallFn          = msg.AfterToolCallFn
	GetMessagesFn            = msg.GetMessagesFn
	StopReason               = msg.StopReason
	Config                   = msg.Config
)

const (
	RoleUser                = msg.RoleUser
	RoleAssistant           = msg.RoleAssistant
	RoleToolResult          = msg.RoleToolResult
	RoleCustom              = msg.RoleCustom
	RoleBranchSummary       = msg.RoleBranchSummary
	RoleCompactionSummary   = msg.RoleCompactionSummary
	ToolExecutionParallel   = msg.ToolExecutionParallel
	ToolExecutionSequential = msg.ToolExecutionSequential
	QueueAll                = msg.QueueAll
	QueueOne                = msg.QueueOne
	StopReasonComplete      = msg.StopReasonComplete
	StopReasonError         = msg.StopReasonError
	StopReasonAborted       = msg.StopReasonAborted
)
