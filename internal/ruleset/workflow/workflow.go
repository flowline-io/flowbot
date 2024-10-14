package workflow

import (
	"errors"
	"time"

	"github.com/flowline-io/flowbot/internal/types"
)

type Rule struct {
	Id           string
	Title        string
	Desc         string
	InputSchema  []types.FormField
	OutputSchema []types.FormField
	Run          func(ctx types.Context, input types.KV) (types.KV, error)
}

type Ruleset []Rule

func (r Ruleset) ProcessRule(ctx types.Context, input types.KV) (types.KV, error) {
	for _, rule := range r {
		if rule.Id == ctx.WorkflowRuleId {
			resultChan := make(chan types.KV, 1)
			errChan := make(chan error, 1)

			// run rule
			go func() {
				result, err := rule.Run(ctx, input) // todo timeout context
				if err != nil {
					errChan <- err
				} else {
					resultChan <- result
				}
			}()

			// 10 minute timeout
			select {
			case result := <-resultChan:
				return result, nil
			case err := <-errChan:
				return types.KV{}, err
			case <-time.After(10 * time.Minute):
				return types.KV{}, errors.New("run timeout")
			}
		}
	}
	return types.KV{}, nil
}
