package types

type GroupEvent int

const (
	GroupEventUnknown GroupEvent = iota
	GroupEventJoin
	GroupEventExit
	GroupEventReceive
)

const (
	MessageSendEvent  = "message:send"
	InstructPushEvent = "instruct:push"
)

type Message struct {
	Platform string
	Topic    string
	Payload  MsgPayload
}
