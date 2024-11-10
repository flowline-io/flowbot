package page

import (
	"fmt"
	"net/http"

	"github.com/flowline-io/flowbot/pkg/page"
	"github.com/flowline-io/flowbot/pkg/types"
)

type Rule struct {
	Id string
	UI func(ctx types.Context, flag string) (*types.UI, error)
}

type Ruleset []Rule

func (r Ruleset) ProcessPage(ctx types.Context, flag string) (string, error) {
	for _, rule := range r {
		if rule.Id == ctx.PageRuleId {
			ui, err := rule.UI(ctx, flag)
			if err != nil {
				return "", err
			}
			return page.Render(ui), nil
		}
	}
	return "", fmt.Errorf("%d not found", http.StatusNotFound)
}
