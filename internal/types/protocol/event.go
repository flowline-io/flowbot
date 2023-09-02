package protocol

const (
	ConnectEvent      = "meta.connect"
	HeartbeatEvent    = "meta.heartbeat"
	StatusUpdateEvent = "meta.status_update"
)

type Event struct {
	Id         string  `json:"id"`
	Time       float64 `json:"time"`
	Type       string  `json:"type"`
	DetailType string  `json:"detail_type"`
	SubType    string  `json:"sub_type"`

	Self       Self             `json:"self,omitempty"`
	MessageId  string           `json:"message_id,omitempty"`
	Message    []MessageSegment `json:"message,omitempty"`
	AltMessage string           `json:"alt_message,omitempty"`
	UserId     string           `json:"user_id,omitempty"`
}

type Self struct {
	Platform string `json:"platform"`
	UserId   string `json:"user_id"`
}
