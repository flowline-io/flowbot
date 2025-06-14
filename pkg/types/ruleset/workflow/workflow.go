package workflow

import (
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"github.com/flowline-io/flowbot/pkg/types"
)

const defaultRunTimeout = 10 * time.Minute

type Rule struct {
	Id          string
	Title       string
	Description string
	Run         func(ctx types.Context, input types.KV) (types.KV, error)
}

func (r Rule) ID() string {
	return r.Id
}

func (r Rule) TYPE() types.RulesetType {
	return types.WorkflowRule
}

type Ruleset []Rule

// ProcessRule processes a specific rule within the Ruleset based on the provided context and input.
// It returns the result of the rule execution or an error if the execution fails or times out.
func (r Ruleset) ProcessRule(ctx types.Context, input types.KV) (types.KV, error) {
	var rule Rule
	for _, item := range r {
		if item.Id == ctx.WorkflowRuleId {
			rule = item
			break
		}
	}
	if rule.Id == "" {
		return nil, nil
	}

	// Ensure the context has a deadline; if not, set a timeout of 10 minutes.
	ctx.SetTimeout(defaultRunTimeout)
	defer ctx.Cancel()

	resultCh := make(chan types.KV, 1)
	errorCh := make(chan error, 1)

	// Start a goroutine to execute the rule.
	go func() {
		defer func() {
			if r := recover(); r != nil { // revive:disable
				_, _ = os.Stderr.WriteString(fmt.Sprintf("panic: %v\n%s\n", r, debug.Stack())) //nolint:errcheck // This will never fail
				errorCh <- fmt.Errorf("recover: %v", r)
			}
		}()

		result, err := rule.Run(ctx, input)
		if err != nil {
			errorCh <- err
		} else {
			resultCh <- result
		}
	}()

	// Wait for the result, error, or context timeout.
	select {
	case result := <-resultCh:
		return result, nil
	case err := <-errorCh:
		return types.KV{}, err
	case <-ctx.Context().Done():
		return types.KV{}, ctx.Context().Err()
	}
}
