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
	// form id
	FormId string
	// form Rule id
	FormRuleId string
	// agent
	CollectId string
	// agent
	AgentVersion int
	// page rule id
	PageRuleId string
	// workflow rule id
	WorkflowRuleId string
	// llm tool rule id
	ToolRuleId string
	// event rule id
	EventId string
}

func (c *Context) Context() context.Context {
	return c.ctx
}

func (c *Context) SetTimeout(timeout time.Duration) {
	if c.ctx == nil {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		c.ctx = ctx
		c.cancel = cancel
		return
	}
	if _, ok := c.ctx.Deadline(); !ok {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		c.ctx = ctx
		c.cancel = cancel
		return
	}
}

func (c *Context) Cancel() context.CancelFunc {
	return c.cancel
}
