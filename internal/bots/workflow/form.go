package workflow

import (
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/form"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/workflow"
	jsoniter "github.com/json-iterator/go"
	"gopkg.in/yaml.v3"
)

const (
	runOneTaskFormID    = "run_one_task"
	createOneTaskFormID = "create_one_task"
)

var formRules = []form.Rule{
	{
		Id:         runOneTaskFormID,
		Title:      "Run one task",
		IsLongTerm: true,
		Field: []types.FormField{
			{
				Key:         "input",
				Type:        types.FormFieldTextarea,
				ValueType:   types.FormFieldValueString,
				Value:       "",
				Label:       "Input",
				Placeholder: "Input params",
				Rule:        "required",
			},
			{
				Key:         "task",
				Type:        types.FormFieldTextarea,
				ValueType:   types.FormFieldValueString,
				Value:       "",
				Label:       "Task",
				Placeholder: "Input task",
				Rule:        "required",
			},
		},
		Handler: func(ctx types.Context, values types.KV) types.MsgPayload {
			inputData, _ := values.String("input")
			taskData, _ := values.String("task")
			var input map[string]interface{}
			err := yaml.Unmarshal([]byte(inputData), &input)
			if err != nil {
				return types.TextMsg{Text: fmt.Sprintf("error: %s", err)}
			}
			var task types.WorkflowTask
			err = yaml.Unmarshal([]byte(taskData), &task)
			if err != nil {
				return types.TextMsg{Text: fmt.Sprintf("error: %s", err)}
			}

			var botName, ruleId string
			if task.Action == "" {
				return types.TextMsg{Text: "action empty"}
			}
			action := strings.Split(task.Action, "@")
			if len(action) != 2 {
				return types.TextMsg{Text: "action invalid"}
			}
			ruleId = action[0]
			botName = action[1]

			var botHandler bots.Handler
			for name, handler := range bots.List() {
				if botName != name {
					continue
				}
				for _, item := range handler.Rules() {
					switch v := item.(type) {
					case []workflow.Rule:
						for _, rule := range v {
							if rule.Id == ruleId {
								botHandler = handler
							}
						}
					}
				}
			}
			if botHandler == nil {
				return types.TextMsg{Text: "bot handler not found"}
			}
			ctx.WorkflowRuleId = ruleId
			output, err := botHandler.Workflow(ctx, input)
			if err != nil {
				return types.TextMsg{Text: fmt.Sprintf("error: %s", err)}
			}

			result, err := jsoniter.Marshal(output)
			if err != nil {
				return types.TextMsg{Text: fmt.Sprintf("error: %s", err)}
			}

			return types.TextMsg{Text: string(result)}
		},
	},
	{
		Id:         createOneTaskFormID,
		Title:      "Create one task",
		IsLongTerm: true,
		Field: []types.FormField{
			{
				Key:         "script",
				Type:        types.FormFieldTextarea,
				ValueType:   types.FormFieldValueString,
				Value:       "",
				Label:       "Script",
				Placeholder: "Input script",
				Rule:        "required",
			},
		},
		Handler: func(ctx types.Context, values types.KV) types.MsgPayload {
			inputData, _ := values.String("script")
			var input map[string]interface{}
			err := yaml.Unmarshal([]byte(inputData), &input)
			if err != nil {
				return types.TextMsg{Text: fmt.Sprintf("error: %s", err)}
			}

			wf, triggers, dag, err := ParseYamlWorkflow(inputData)
			if err != nil {
				return types.TextMsg{Text: fmt.Sprintf("error: %s", err)}
			}

			wf.UID = ctx.AsUser.String()
			wf.Topic = ctx.Topic
			script := new(model.WorkflowScript)
			script.Lang = model.WorkflowScriptYaml
			script.Code = inputData
			_, err = store.Database.CreateWorkflow(wf, script, dag, triggers)
			if err != nil {
				return types.TextMsg{Text: fmt.Sprintf("error: %s", err)}
			}

			return types.KVMsg{
				"id": wf.ID,
			}
		},
	},
}
