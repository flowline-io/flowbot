package types

const (
	MessageSendEvent  = "message:send"
	InstructPushEvent = "instruct:push"
	BotRunEvent       = "bot:event"
	ModuleRunEvent    = BotRunEvent
)

const (
	ExampleBotEventID         = "example"
	TaskCreateBotEventID      = "creteTask"
	BookmarkArchiveBotEventID = "archiveBookmark"
	BookmarkCreateBotEventID  = "createBookmark"
	ArchiveBoxAddBotEventID   = "archiveBoxAdd"
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

// ModuleEvent is the module-era name for the legacy BotEvent.
type ModuleEvent = BotEvent

const (
	EventBookmarkCreated  = "bookmark.created"
	EventBookmarkArchived = "bookmark.archived"

	EventArchiveItemCreated = "archive.item.created"

	EventReaderEntryStarred = "reader.entry.starred"
	EventReaderEntryRead    = "reader.entry.read"

	EventKanbanTaskCreated   = "kanban.task.created"
	EventKanbanTaskCompleted = "kanban.task.completed"

	EventInfraHostDown = "infra.host.down"
	EventInfraHostUp   = "infra.host.up"
)

// DataEvent is the durable business event contract emitted by ability write operations.
type DataEvent struct {
	EventID        string `json:"event_id"`
	EventType      string `json:"event_type"`
	Source         string `json:"source"`
	Capability     string `json:"capability"`
	Operation      string `json:"operation"`
	Backend        string `json:"backend"`
	App            string `json:"app"`
	EntityID       string `json:"entity_id"`
	IdempotencyKey string `json:"idempotency_key"`
	UID            string `json:"uid"`
	Topic          string `json:"topic"`
	Data           KV     `json:"data"`
}
