package types

import (
	"context"
	"time"
)

type Context struct {
	// ctx is the context
	ctx context.Context
	// cancel function
	cancel context.CancelFunc

	// Message ID denormalized
	Id string
	// chat platform
	Platform string
	// channel or group
	Topic string
	// Sender's UserId as string.
	AsUser Uid

	// form Rule id
	FormRuleId string
	// form id
	FormId string

	// agent rule id
	CollectRuleId string
	// agent
	AgentVersion int

	// page rule id
	PageRuleId string

	// workflow rule id
	WorkflowRuleId string

	// llm tool rule id
	ToolRuleId string

	// event rule id
	EventRuleId string

	// webhook rule id
	WebhookRuleId string
	// HTTP method
	Method string
	// HTTP headers
	Headers map[string][]string
}

func (c *Context) Context() context.Context {
	return c.ctx
}

func (c *Context) SetTimeout(timeout time.Duration) {
	// If there is an existing cancel function, call it first to avoid resource leaks
	if c.cancel != nil {
		c.cancel()
	}

	// If the context is nil, create a new context with timeout
	if c.ctx == nil {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		c.ctx = ctx
		c.cancel = cancel
		return
	}

	// If the context already exists but has no deadline, create a new context with timeout based on the existing context
	if _, ok := c.ctx.Deadline(); !ok {
		ctx, cancel := context.WithTimeout(c.ctx, timeout)
		c.ctx = ctx
		c.cancel = cancel
		return
	}
}

func (c *Context) Cancel() context.CancelFunc {
	return c.cancel
}
