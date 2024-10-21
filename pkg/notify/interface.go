package notify

import "github.com/flowline-io/flowbot/internal/types"

type Notifyer interface {
	// Protocol Define protocol
	Protocol() string
	// Templates Define object templates
	Templates() []string
	// Send notify
	Send(tokens types.KV, message Message) error
}

type Priority int32

const (
	Low Priority = iota + 1
	Moderate
	Normal
	High
	Emergency
)

type Message struct {
	Title    string
	Body     string
	Url      string
	Priority Priority
}
