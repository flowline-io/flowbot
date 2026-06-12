package model

import "github.com/flowline-io/flowbot/pkg/agent/msg"

// Router selects between chat and tool models for dual-model strategies.
type Router struct {
	ChatModel string
	ToolModel string
}

// NewRouter creates a router with explicit chat and tool model names.
func NewRouter(chatModel, toolModel string) *Router {
	return &Router{ChatModel: chatModel, ToolModel: toolModel}
}

// Select chooses the model for the next provider request.
func (r *Router) Select(afterToolExecution bool) string {
	if afterToolExecution && r.ToolModel != "" {
		return r.ToolModel
	}
	if r.ChatModel != "" {
		return r.ChatModel
	}
	return r.ToolModel
}

// ApplyToContext updates the agent context model field using router defaults.
func (r *Router) ApplyToContext(ctx *msg.Context, afterToolExecution bool) {
	if ctx == nil || r == nil {
		return
	}
	ctx.ModelName = r.Select(afterToolExecution)
}

// ApplyDefaultRouter injects dual-model PrepareNextTurn when chat and tool models are set.
func ApplyDefaultRouter(cfg msg.Config) msg.Config {
	if cfg.PrepareNextTurn != nil || cfg.ChatModel == "" || cfg.ToolModel == "" {
		return cfg
	}
	router := NewRouter(cfg.ChatModel, cfg.ToolModel)
	cfg.PrepareNextTurn = router.PrepareNextTurnHook()
	if cfg.ModelName == "" {
		cfg.ModelName = cfg.ChatModel
	}
	return cfg
}

// PrepareNextTurnHook returns a turn-boundary hook that routes to chat or tool models.
func (r *Router) PrepareNextTurnHook() msg.PrepareNextTurnFn {
	return func(turn msg.TurnContext) (*msg.TurnUpdate, error) {
		ctx := cloneContext(turn.Context)
		r.ApplyToContext(ctx, len(turn.ToolResults) > 0)
		return &msg.TurnUpdate{Context: ctx, ModelName: ctx.ModelName}, nil
	}
}

func cloneContext(src *msg.Context) *msg.Context {
	if src == nil {
		return &msg.Context{}
	}
	clone := *src
	clone.Messages = append([]msg.AgentMessage(nil), src.Messages...)
	return &clone
}
