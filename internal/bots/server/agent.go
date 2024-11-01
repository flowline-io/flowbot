package server

import (
	"github.com/flowline-io/flowbot/internal/ruleset/collect"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
)

const (
	StatsCollectID = "stats_collect"
)

var collectRules = []collect.Rule{
	{
		Id:   StatsCollectID,
		Help: "upload server status",
		Args: []string{"cpu", "memory", "info"},
		Handler: func(ctx types.Context, content types.KV) types.MsgPayload {
			j := types.KV{}
			err := j.Scan(content)
			if err != nil {
				return nil
			}
			// alert

			// store
			err = store.Database.DataSet(ctx.AsUser, ctx.Topic, "stats", j)
			if err != nil {
				flog.Error(err)
				return nil
			}
			return nil
		},
	},
}
