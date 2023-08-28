package config

import (
	"encoding/json"
)

var Server struct {
	Log      Log      `json:"log"`
	Workflow Workflow `json:"workflow"`
}

type Log struct {
	Level string `json:"level"`
}

type Workflow struct {
	Worker int `json:"worker"`
}

func Load(raw json.RawMessage) error {
	return json.Unmarshal(raw, &Server)
}
