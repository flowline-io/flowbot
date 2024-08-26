package dev

import (
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/internal/ruleset/workflow"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/event"
	"github.com/flowline-io/flowbot/pkg/flog"
)

const (
	inWorkflowActionID      = "in"
	addWorkflowActionID     = "add"
	outWorkflowActionID     = "out"
	errorWorkflowActionID   = "error"
	messageWorkflowActionID = "message"
)

var workflowRules = []workflow.Rule{
	{
		Id:           inWorkflowActionID,
		Title:        "in",
		Desc:         "",
		InputSchema:  nil,
		OutputSchema: nil,
		Run: func(ctx types.Context, input types.KV) (types.KV, error) {
			return types.KV{"a": 1, "b": 1}, nil
		},
	},
	{
		Id:           addWorkflowActionID,
		Title:        "add",
		Desc:         "",
		InputSchema:  nil,
		OutputSchema: nil,
		Run: func(ctx types.Context, input types.KV) (types.KV, error) {
			a, _ := input.Int64("a")
			b, _ := input.Int64("b")
			return types.KV{"value": a + b}, nil
		},
	},
	{
		Id:           outWorkflowActionID,
		Title:        "out",
		Desc:         "",
		InputSchema:  nil,
		OutputSchema: nil,
		Run: func(ctx types.Context, input types.KV) (types.KV, error) {
			flog.Debug("%s => %+v", outWorkflowActionID, input)
			return nil, nil
		},
	},
	{
		Id:           errorWorkflowActionID,
		Title:        "error",
		Desc:         "",
		InputSchema:  nil,
		OutputSchema: nil,
		Run: func(ctx types.Context, input types.KV) (types.KV, error) {
			return nil, fmt.Errorf("workflow run error %s", time.Now().Format(time.DateTime))
		},
	},
	{
		Id:           messageWorkflowActionID,
		Title:        "message",
		Desc:         "",
		InputSchema:  nil,
		OutputSchema: nil,
		Run: func(ctx types.Context, input types.KV) (types.KV, error) {
			text, _ := input.String("text")
			if text == "" {
				return nil, fmt.Errorf("%s step, empty text", messageWorkflowActionID)
			}
			return nil, event.SendMessage(ctx.AsUser.String(), ctx.Topic, types.TextMsg{Text: text})
		},
	},
}
