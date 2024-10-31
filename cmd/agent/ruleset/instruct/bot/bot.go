package bot

import "github.com/flowline-io/flowbot/internal/types"

type Executor struct {
	Flag string
	Run  func(app any, window any, data types.KV) error
}

var DoInstruct = map[string][]Executor{
	"dev":       dev,
	"clipboard": clipboard,
	"url":       url,
}
