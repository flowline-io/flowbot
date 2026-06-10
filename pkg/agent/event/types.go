package event

// Type identifies lifecycle events emitted by the agent loop.
type Type string

const (
	TypeAgentStart          Type = "agent_start"
	TypeTurnStart           Type = "turn_start"
	TypeMessageStart        Type = "message_start"
	TypeMessageUpdate       Type = "message_update"
	TypeMessageEnd          Type = "message_end"
	TypeToolExecutionStart  Type = "tool_execution_start"
	TypeToolExecutionUpdate Type = "tool_execution_update"
	TypeToolExecutionEnd    Type = "tool_execution_end"
	TypeTurnEnd             Type = "turn_end"
	TypeAgentEnd            Type = "agent_end"
)

// Event is a structured lifecycle notification for UI and harness consumers.
type Event struct {
	Type        Type
	Message     any
	ToolCall    any
	ToolResult  any
	ToolResults any
	Messages    any
	TextDelta   string
	Update      string
	Err         error
}

// Handler processes agent lifecycle events sequentially.
type Handler func(Event) error
