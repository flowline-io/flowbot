package mtg

import (
	"context"
	"fmt"
	"github.com/flowline-io/flowbot/internal/ruleset/command"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/providers/scryfall"
)

var commandRules = []command.Rule{
	{
		Define: "search [string]",
		Help:   `Search cards.`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			keyword, _ := tokens[1].Value.String()
			provider := scryfall.NewScryfall()
			result, err := provider.CardsSearch(context.Background(), fmt.Sprintf("%s lang:zhs", keyword))
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: "search error"}
			}
			if len(result) == 0 {
				return types.TextMsg{Text: "empty"}
			}
			limit := 0
			var cards []types.CardMsg
			for _, card := range result {
				if limit >= 10 {
					break
				}
				name := card.PrintedName
				if name == "" {
					name = card.Name
				}
				cards = append(cards, types.CardMsg{
					Name: name,
					URI:  card.ScryfallURI,
				})
				limit++
			}
			return types.CardListMsg{
				Cards: cards,
			}
		},
	},
}
