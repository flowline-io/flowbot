package hooks

import "github.com/flowline-io/flowbot/pkg/agent/msg"

// ChainTransformContext runs inner before outer so base transforms apply before hook transforms.
func ChainTransformContext(inner, outer msg.TransformContextFn) msg.TransformContextFn {
	if outer == nil {
		return inner
	}
	if inner == nil {
		return outer
	}
	return func(messages []msg.AgentMessage) ([]msg.AgentMessage, error) {
		transformed, err := inner(messages)
		if err != nil {
			return nil, err
		}
		return outer(transformed)
	}
}

// ChainBeforeToolCall runs inner before outer and preserves the first block result.
func ChainBeforeToolCall(inner, outer msg.BeforeToolCallFn) msg.BeforeToolCallFn {
	if outer == nil {
		return inner
	}
	if inner == nil {
		return outer
	}
	return func(ctx msg.BeforeToolContext) (*msg.BeforeToolResult, error) {
		if inner != nil {
			result, err := inner(ctx)
			if err != nil {
				return nil, err
			}
			if result != nil && result.Block {
				return result, nil
			}
		}
		return outer(ctx)
	}
}

// ChainAfterToolCall runs inner before outer so hook patches apply after base patches.
func ChainAfterToolCall(inner, outer msg.AfterToolCallFn) msg.AfterToolCallFn {
	if outer == nil {
		return inner
	}
	if inner == nil {
		return outer
	}
	return func(ctx msg.AfterToolContext) (*msg.AfterToolResult, error) {
		var base *msg.AfterToolResult
		if inner != nil {
			result, err := inner(ctx)
			if err != nil {
				return nil, err
			}
			base = result
			if result != nil {
				ctx.Result = applyAfterToolPatch(ctx.Result, result)
			}
		}
		hookResult, err := outer(ctx)
		if err != nil {
			return nil, err
		}
		if hookResult == nil {
			return base, nil
		}
		if base != nil && base.Terminate {
			hookResult.Terminate = true
		}
		return hookResult, nil
	}
}

// ChainPrepareNextTurn runs inner before outer at turn boundaries.
func ChainPrepareNextTurn(inner, outer msg.PrepareNextTurnFn) msg.PrepareNextTurnFn {
	if outer == nil {
		return inner
	}
	if inner == nil {
		return outer
	}
	return func(ctx msg.TurnContext) (*msg.TurnUpdate, error) {
		update, err := inner(ctx)
		if err != nil {
			return nil, err
		}
		if update != nil {
			if update.Context != nil {
				ctx.Context = update.Context
			}
			if update.ModelName != "" && ctx.Context != nil {
				ctx.Context.ModelName = update.ModelName
			}
		}
		return outer(ctx)
	}
}

func applyAfterToolPatch(result msg.ToolResultMessage, patch *msg.AfterToolResult) msg.ToolResultMessage {
	if patch == nil {
		return result
	}
	if len(patch.Parts) > 0 {
		result.Parts = append([]msg.ContentPart(nil), patch.Parts...)
	}
	if patch.IsError != nil {
		result.IsError = *patch.IsError
	}
	return result
}
