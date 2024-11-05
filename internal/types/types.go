package types

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/lithammer/shortuuid/v4"
)

type MsgPayload interface {
	Convert() (KV, interface{})
}

type EventPayload struct {
	Typ string
	Src []byte
}

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

func Id() string {
	return shortuuid.New()
}

func AppUrl() string {
	return config.App.Flowbot.URL
}

type DataFilter struct {
	Prefix       *string
	CreatedStart *time.Time
	CreatedEnd   *time.Time
}

// TimeNow returns current wall time in UTC rounded to milliseconds.
func TimeNow() time.Time {
	return time.Now().UTC().Round(time.Millisecond)
}
