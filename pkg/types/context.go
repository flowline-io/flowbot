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

	// TraceCtx carries the OpenTelemetry trace context for span propagation.
	// Set via SetTraceContext before calling SetTimeout to inherit the trace parent.
	TraceCtx context.Context

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

	// page rule id
	PageRuleId string

	// workflow rule id
	WorkflowRuleId string

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
	if c.ctx == nil {
		return context.Background()
	}
	return c.ctx
}

func (c *Context) SetTimeout(timeout time.Duration) {
	// If there is an existing cancel function, call it first to avoid resource leaks
	if c.cancel != nil {
		c.cancel()
	}

	// If the context is nil, create a new context with timeout
	if c.ctx == nil {
		parent := c.TraceCtx
		if parent == nil {
			parent = context.Background()
		}
		ctx, cancel := context.WithTimeout(parent, timeout)
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
