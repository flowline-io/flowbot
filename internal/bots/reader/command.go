package reader

import (
	"fmt"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/providers/miniflux"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
	rssClient "miniflux.app/v2/client"
)

var commandRules = []command.Rule{
	{
		Define: "unread",
		Help:   `Show miniflux unread total`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			endpoint, _ := providers.GetConfig(miniflux.ID, miniflux.EndpointKey)
			apiKey, _ := providers.GetConfig(miniflux.ID, miniflux.ApikeyKey)
			client := miniflux.NewMiniflux(endpoint.String(), apiKey.String())

			result, err := client.GetEntries(&rssClient.Filter{Status: rssClient.EntryStatusUnread, Limit: 1})
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			return types.TextMsg{Text: fmt.Sprintf("unread total: %d", result.Total)}
		},
	},
}
