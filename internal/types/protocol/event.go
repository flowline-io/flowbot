package protocol

const (
	MetaConnectEvent      = "meta.connect"
	MetaHeartbeatEvent    = "meta.heartbeat"
	MetaStatusUpdateEvent = "meta.status_update"

	MessageDirectEvent  = "message.direct"
	MessageGroupEvent   = "message.group"
	MessageChannelEvent = "message.channel"
	MessageCommandEvent = "message.command"

	NoticeFriendIncreaseEvent      = "notice.friend_increase"
	NoticeFriendDecreaseEvent      = "notice.friend_decrease"
	NoticeGroupMemberIncreaseEvent = "notice.group_member_increase"
	NoticeGroupMemberDecreaseEvent = "notice.group_member_decrease"
	NoticeChannelCreateEvent       = "notice.channel_create"
	NoticeChannelDeleteEvent       = "notice.channel_delete"
)

type EventType string

const (
	MetaEventType    EventType = "meta"
	MessageEventType EventType = "message"
	NoticeEventType  EventType = "notice"
	RequestEventType EventType = "request"
)

type Event struct {
	Id         string    `json:"id"`
	Time       int64     `json:"time"`
	Type       EventType `json:"type"`
	DetailType string    `json:"detail_type"`
	Data       any       `json:"data"`
}

type MessageEventData struct {
	Self       Self             `json:"self,omitempty"`
	MessageId  string           `json:"message_id,omitempty"`
	Message    []MessageSegment `json:"message,omitempty"`
	AltMessage string           `json:"alt_message,omitempty"`
	UserId     string           `json:"user_id,omitempty"`

	TopicId   string `json:"topic_id,omitempty"`
	TopicType string `json:"topic_type,omitempty"`

	Forwarded string `json:"forwarded,omitempty"`

	Seq    float64 `json:"seq,omitempty"`
	Option string  `json:"option,omitempty"`
}

type CommandEventData struct {
	Command string `json:"command,omitempty"`
}

type Self struct {
	Platform string `json:"platform"`
	UserId   string `json:"user_id"`
}
