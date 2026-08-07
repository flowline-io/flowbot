package dcg

import (
	"context"
)

// ToolVerdict is the outcome of EvaluateToolCall.
type ToolVerdict struct {
	// Skip is true when the tool is not guarded by dcg.
	Skip bool
	// Block is true when the tool call must not run.
	Block bool
	// Reason explains a block when Block is true.
	Reason string
	// Command is the evaluated command when the tool is guarded.
	Command string
	// RuleID is the matching dcg rule when present.
	RuleID string
	// PackID is the matching dcg pack when present.
	PackID string
}

// EvaluateToolCall extracts a command for tool and runs checker.
// checker nil uses DefaultChecker().
func EvaluateToolCall(ctx context.Context, tool string, args map[string]any, checker Checker) (ToolVerdict, error) {
	command, ok, err := CommandForTool(tool, args)
	if err != nil {
		return ToolVerdict{Block: true, Reason: err.Error()}, nil
	}
	if !ok {
		return ToolVerdict{Skip: true}, nil
	}
	if checker == nil {
		checker = DefaultChecker()
	}
	decision, err := checker.Check(ctx, command)
	if err != nil {
		return ToolVerdict{Block: true, Reason: err.Error(), Command: command}, nil
	}
	if !decision.Allow {
		reason := decision.Reason
		if reason == "" {
			reason = ReasonBlocked
		}
		return ToolVerdict{
			Block:   true,
			Reason:  reason,
			Command: command,
			RuleID:  decision.RuleID,
			PackID:  decision.PackID,
		}, nil
	}
	return ToolVerdict{Command: command}, nil
}
