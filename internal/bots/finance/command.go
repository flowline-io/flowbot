package finance

import (
	"context"
	"fmt"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/providers/wallos"

	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/providers/doctorxiong"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
)

var commandRules = []command.Rule{
	{
		Define: `fund [string]`,
		Help:   `Get fund`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			code, _ := tokens[1].Value.String()

			reply, err := doctorxiong.GetFund(context.Background(), code)
			if err != nil {
				return nil
			}

			if reply.Name != "" {
				var xAxis []string
				var series []float64
				if len(reply.NetWorthDataDate) == 0 {
					xAxis = reply.MillionCopiesIncomeDataDate
					series = reply.MillionCopiesIncomeDataIncome
				} else {
					xAxis = reply.NetWorthDataDate
					series = reply.NetWorthDataUnit
				}

				title := fmt.Sprintf("Fund %s (%s)", reply.Name, reply.Code)
				return bots.StorePage(ctx, model.PageChart, title, types.ChartMsg{
					Title:    title,
					SubTitle: "Data for the last 90 days",
					XAxis:    xAxis,
					Series:   series,
				})
			}

			return types.TextMsg{Text: "failed"}
		},
	},
	{
		Define: `stock [string]`,
		Help:   `Get stock`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			code, _ := tokens[1].Value.String()

			reply, err := doctorxiong.GetStock(context.Background(), code)
			if err != nil {
				return nil
			}

			return types.InfoMsg{
				Title: fmt.Sprintf("Stock %s", code),
				Model: reply,
			}
		},
	},
	{
		Define: `wallos`,
		Help:   `Get wallos subscriptions`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			endpoint, _ := providers.GetConfig(wallos.ID, wallos.EndpointKey)
			apiKey, _ := providers.GetConfig(wallos.ID, wallos.ApikeyKey)

			client := wallos.NewWallos(endpoint.String(), apiKey.String())
			list, err := client.GetSubscriptions()
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			return types.InfoMsg{
				Title: "Wallos Subscriptions",
				Model: list,
			}
		},
	},
}
