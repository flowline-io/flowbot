package types

const ApiVersion = 1

type Action string

const (
	PullAction    Action = "pull"
	AckAction     Action = "ack"
	OnlineAction  Action = "online"
	OfflineAction Action = "offline"
	MessageAction Action = "message"
)

type AgentData struct {
	Action  Action `json:"action" validate:"required,oneof=pull ack online offline message"`
	Version int    `json:"version" validate:"gte=0"`
	Content KV     `json:"content"`
}

const (
	// Assistant is the role of an assistant, means the message is returned by ChatModel.
	Assistant string = "assistant"
	// User is the role of a user, means the message is a user message.
	User string = "user"
	// System is the role of a system, means the message is a system message.
	System string = "system"
	// Tool is the role of a tool, means the message is a tool call output.
	Tool string = "tool"
)

type Executor struct {
	Flag string
	Run  func(data KV) error
}
