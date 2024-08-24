package types

import (
	"time"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/lithammer/shortuuid/v4"
)

type MsgPayload interface {
	Convert() (KV, interface{})
}

type EventPayload struct {
	Type   string
	Params KV
}

type Context struct {
	// Message ID denormalized
	Id string
	// chat platform
	Platform string
	// channel or group
	Topic string
	// Sender's UserId as string.
	AsUser Uid
	// OAuth token
	Token string
	// form id
	FormId string
	// form Rule id
	FormRuleId string
	// agent
	AgentId string
	// agent
	AgentVersion int
	// page rule id
	PageRuleId string
	// workflow rule id
	WorkflowRuleId string
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

type SendFunc func(topic string, uid Uid, out MsgPayload, option ...interface{})

// TimeNow returns current wall time in UTC rounded to milliseconds.
func TimeNow() time.Time {
	return time.Now().UTC().Round(time.Millisecond)
}
