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
