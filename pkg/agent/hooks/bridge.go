package hooks

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/agent/msg"
)

// BridgeConfig composes registry hook handlers onto an existing loop configuration.
func BridgeConfig(runCtx context.Context, reg *Registry, base msg.Config) msg.Config {
	if reg == nil || !reg.HasLoopHandlers() {
		return base
	}
	if runCtx == nil {
		runCtx = context.Background()
	}
	base.TransformContext = ChainTransformContext(base.TransformContext, reg.transformContextFn(runCtx))
	base.BeforeToolCall = ChainBeforeToolCall(base.BeforeToolCall, reg.beforeToolCallFn(runCtx))
	base.AfterToolCall = ChainAfterToolCall(base.AfterToolCall, reg.afterToolCallFn(runCtx))
	return base
}

// MergeHookFields copies hook-related callbacks from src onto dst without replacing queue drains.
func MergeHookFields(dst, src *msg.Config) {
	if dst == nil || src == nil {
		return
	}
	dst.TransformContext = src.TransformContext
	dst.BeforeToolCall = src.BeforeToolCall
	dst.AfterToolCall = src.AfterToolCall
	dst.PrepareNextTurn = src.PrepareNextTurn
}

func (r *Registry) transformContextFn(runCtx context.Context) msg.TransformContextFn {
	return func(messages []msg.AgentMessage) ([]msg.AgentMessage, error) {
		if err := runCtx.Err(); err != nil {
			return nil, err
		}
		return r.EmitContext(runCtx, messages)
	}
}

func (r *Registry) beforeToolCallFn(runCtx context.Context) msg.BeforeToolCallFn {
	return func(ctx msg.BeforeToolContext) (*msg.BeforeToolResult, error) {
		if err := runCtx.Err(); err != nil {
			return nil, err
		}
		result, err := r.EmitToolCall(runCtx, ToolCallEvent{
			Assistant: ctx.Assistant,
			ToolCall:  ctx.ToolCall,
			Args:      ctx.Args,
			Context:   ctx.Context,
		})
		if err != nil {
			return nil, err
		}
		if result == nil {
			return nil, nil
		}
		return &msg.BeforeToolResult{Block: result.Block, Reason: result.Reason}, nil
	}
}

func (r *Registry) afterToolCallFn(runCtx context.Context) msg.AfterToolCallFn {
	return func(ctx msg.AfterToolContext) (*msg.AfterToolResult, error) {
		if err := runCtx.Err(); err != nil {
			return nil, err
		}
		result, err := r.EmitToolResult(runCtx, ToolResultEvent{
			Assistant: ctx.Assistant,
			ToolCall:  ctx.ToolCall,
			Args:      ctx.Args,
			Result:    ctx.Result,
			Context:   ctx.Context,
		})
		if err != nil {
			return nil, err
		}
		if result == nil {
			return nil, nil
		}
		return &msg.AfterToolResult{
			Parts:     result.Parts,
			IsError:   result.IsError,
			Terminate: result.Terminate,
		}, nil
	}
}
