package dev

import (
	"fmt"
	"github.com/flowline-io/flowbot/internal/ruleset/workflow"
	"github.com/flowline-io/flowbot/internal/types"
)

const (
	inWorkflowActionID  = "in_workflow_action"
	addWorkflowActionID = "add_workflow_action"
	outWorkflowActionID = "out_workflow_action"
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
			fmt.Println("=========>", input)
			return nil, nil
		},
	},
}
