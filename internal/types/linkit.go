package types

type Action string

const (
	Info  Action = "info"
	Pull  Action = "pull"
	Agent Action = "agent"
	Bots  Action = "bots"
	Help  Action = "help"
	Ack   Action = "ack"
)

type LinkData struct {
	Action  Action `json:"action"`
	Version int    `json:"version"`
	Content KV     `json:"content"`
}
