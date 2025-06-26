package types

import (
	"fmt"
)

const ApiVersion = 1

type Action string

const (
	PullAction    Action = "pull"
	CollectAction Action = "collect"
	AckAction     Action = "ack"
	OnlineAction  Action = "online"
	OfflineAction Action = "offline"
	MessageAction Action = "message"
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

const (
	// Assistant is the role of an assistant, means the message is returned by ChatModel.
	Assistant string = "assistant"
	// User is the role of a user, means the message is a user message.
	User string = "user"
	// System is the role of a system, means the message is a system message.
	System string = "system"
	// Tool is the role of a tool, means the message is a tool call output.
	Tool string = "tool"
)

type Executor struct {
	Flag string
	Run  func(data KV) error
}

var DoInstruct = map[string][]Executor{}

func InstructRegister(name string, list []Executor) {
	if DoInstruct == nil {
		DoInstruct = make(map[string][]Executor)
	}
	if _, dup := DoInstruct[name]; dup {
		panic(fmt.Sprintf("Register: called twice for instruct bot %s", name))
	}
	DoInstruct[name] = list
}
