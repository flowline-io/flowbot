package types

const ApiVersion = 1

type Action string

const (
	Pull    Action = "pull"
	Collect Action = "collect"
	Ack     Action = "ack"
)

type AgentData struct {
	Action  Action `json:"action"`
	Version int    `json:"version"`
	Content KV     `json:"content"`
}

type CollectData struct {
	Id      string `json:"id"`
	Content KV     `json:"content"`
}
