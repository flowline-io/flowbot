package dev

import (
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/workflow"
)

const (
	inWorkflowActionID    = "in"
	addWorkflowActionID   = "add"
	outWorkflowActionID   = "out"
	errorWorkflowActionID = "error"
)

var workflowRules = []workflow.Rule{
	{
		Id:           inWorkflowActionID,
		Title:        "in",
		Desc:         "return {a, b}",
		InputSchema:  nil,
		OutputSchema: nil,
		Run: func(ctx types.Context, input types.KV) (types.KV, error) {
			return types.KV{"a": 1, "b": 1}, nil
		},
	},
	{
		Id:           addWorkflowActionID,
		Title:        "add",
		Desc:         "a + b",
		InputSchema:  nil,
		OutputSchema: nil,
		Run: func(ctx types.Context, input types.KV) (types.KV, error) {
			a, _ := input.Int64("a")
			b, _ := input.Int64("b")
			return types.KV{"value": add(a, b)}, nil
		},
	},
	{
		Id:           outWorkflowActionID,
		Title:        "out",
		Desc:         "print debug log",
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
		Desc:         "return error",
		InputSchema:  nil,
		OutputSchema: nil,
		Run: func(ctx types.Context, input types.KV) (types.KV, error) {
			return nil, fmt.Errorf("workflow run error %s", time.Now().Format(time.DateTime))
		},
	},
}
