package types

const ApiVersion = 1

type Action string

const (
	Info    Action = "info"
	Pull    Action = "pull"
	Collect Action = "collect"
	Bots    Action = "bots"
	Help    Action = "help"
	Ack     Action = "ack"
)

type FlowkitData struct {
	Action  Action `json:"action"`
	Version int    `json:"version"`
	Content KV     `json:"content"`
}
