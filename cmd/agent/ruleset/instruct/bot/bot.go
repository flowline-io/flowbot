package bot

import "github.com/flowline-io/flowbot/pkg/types"

type Executor struct {
	Flag string
	Run  func(data types.KV) error
}

var DoInstruct = map[string][]Executor{
	"dev":       dev,
	"clipboard": clipboard,
	"url":       url,
}
