package dev

import (
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/event"
)

var eventRules = []event.Rule{
	{
		Id: types.ExampleBotEventID,
		Handler: func(ctx types.Context, param types.KV) error {
			flog.Info("[event] run %s with param %v", types.ExampleBotEventID, param)
			return nil
		},
	},
}
