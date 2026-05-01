package dev

import (
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/collect"
)

const (
	ImportCollectID = "import_collect"
)

var collectRules = []collect.Rule{
	{
		Id:   ImportCollectID,
		Help: "collect example",
		Args: []string{},
		Handler: func(ctx types.Context, content types.KV) types.MsgPayload {
			return nil
		},
	},
}
