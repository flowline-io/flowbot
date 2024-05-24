package finance

import (
	"context"
	"fmt"
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/ruleset/command"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/providers/doctorxiong"
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
				if reply.NetWorthDataDate == nil || len(reply.NetWorthDataDate) == 0 {
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
}
