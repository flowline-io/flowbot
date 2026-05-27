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

	// HTTP method
	Method string
	// HTTP headers
	Headers map[string][]string
}

// Context returns the underlying context.Context.
//
// Priority: c.ctx (set by SetTimeout or SetContext) > c.TraceCtx > context.Background().
// This ensures that trace context from HTTP requests is not silently dropped when SetTimeout
// has not been called.
func (c *Context) Context() context.Context {
	if c.ctx != nil {
		return c.ctx
	}
	if c.TraceCtx != nil {
		return c.TraceCtx
	}
	return context.Background()
}

// SetContext stores ctx as both the internal context and the trace context.
// Use this when you have a traced context (e.g., from an HTTP request or event message)
// but do not yet need a deadline.
func (c *Context) SetContext(ctx context.Context) {
	c.TraceCtx = ctx
	c.ctx = ctx
}

// SetTraceContext stores traceCtx in the TraceCtx field without modifying the internal context.
// Call this before SetTimeout to ensure the timeout context inherits the trace parent.
func (c *Context) SetTraceContext(traceCtx context.Context) {
	c.TraceCtx = traceCtx
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
