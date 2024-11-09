package search

import (
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/ruleset/collect"
)

const (
	ExampleCollectID = "search_example_collect"
)

var collectRules = []collect.Rule{
	{
		Id: ExampleCollectID,
		Handler: func(ctx types.Context, content types.KV) types.MsgPayload {
			return nil
		},
	},
}
