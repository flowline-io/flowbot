package types

const ApiVersion = 1

type Action string

const (
	Pull    Action = "pull"
	Collect Action = "collect"
	Ack     Action = "ack"
	Online  Action = "online"
	Offline Action = "offline"
)

type AgentData struct {
	Action  Action `json:"action"`
	Version int    `json:"version"`
	Content KV     `json:"content"`
}

type CollectData struct {
	Id      string `json:"id"`
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
