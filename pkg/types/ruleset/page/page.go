package page

import (
	"fmt"
	"net/http"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/page"
	"github.com/flowline-io/flowbot/pkg/types"
)

type Rule struct {
	Id string
	UI func(ctx types.Context, flag string, args types.KV) (*types.UI, error)
}

type Ruleset []Rule

func (r Ruleset) ProcessPage(ctx types.Context, flag string, args types.KV) (string, error) {
	for _, rule := range r {
		if rule.Id == ctx.PageRuleId {
			p, err := store.Database.ParameterGet(flag)
			if err != nil {
				return "", err
			}
			ui, err := rule.UI(ctx, flag, args)
			if err != nil {
				return "", err
			}
			ui.Global = types.KV(p.Params)
			ui.ExpiredAt = p.ExpiredAt
			return page.Render(ui), nil
		}
	}
	return "", fmt.Errorf("%d not found", http.StatusNotFound)
}
