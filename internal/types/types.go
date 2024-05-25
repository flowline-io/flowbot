package types

import (
	"github.com/flowline-io/flowbot/pkg/config"
	jsoniter "github.com/json-iterator/go"
	"github.com/lithammer/shortuuid/v3"
	"time"
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
	// Un-routable (original) topic name denormalized from XXX.Topic.
	Original string
	// Routable (expanded) topic name.
	RcptTo string
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

type QueuePayload struct {
	RcptTo string `json:"rcpt_to"`
	Uid    string `json:"uid"`
	Type   string `json:"type"`
	Msg    []byte `json:"msg"`
}

func ConvertQueuePayload(rcptTo string, uid string, msg MsgPayload) (QueuePayload, error) {
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	data, err := json.Marshal(msg)
	if err != nil {
		return QueuePayload{}, err
	}
	return QueuePayload{
		RcptTo: rcptTo,
		Uid:    uid,
		Type:   Tye(msg),
		Msg:    data,
	}, nil
}

type DataFilter struct {
	Prefix       *string
	CreatedStart *time.Time
	CreatedEnd   *time.Time
}

type SendFunc func(rcptTo string, uid Uid, out MsgPayload, option ...interface{})

func WithContext(ctx Context) Context {
	return ctx
}

// TimeNow returns current wall time in UTC rounded to milliseconds.
func TimeNow() time.Time {
	return time.Now().UTC().Round(time.Millisecond)
}
