package types

const (
	MessageSendEvent  = "message:send"
	InstructPushEvent = "instruct:push"
)

type Message struct {
	Platform string
	Topic    string
	Payload  MsgPayload
}
