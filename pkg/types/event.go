package types

const (
	MessageSendEvent  = "message:send"
	InstructPushEvent = "instruct:push"
	BotRunEvent       = "bot:event"
)

const (
	ExampleBotEventID         = "example"
	TaskCreateBotEventID      = "creteTask"
	BookmarkArchiveBotEventID = "archiveBookmark"
	BookmarkCreateBotEventID  = "createBookmark"
)

type Message struct {
	Platform string
	Topic    string
	Payload  EventPayload
}

type EventPayload struct {
	Typ string
	Src []byte
}

type BotEvent struct {
	EventName string
	Uid       string
	Topic     string
	Param     KV
}
