package notify

type Notifyer interface {
	// Define protocol
	Protocol() string
	// Define object templates
	Templates() []string
	// Parse and load config template
	ParseTokens(line string) error
	// Send notify
	Send(message Message) error
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
