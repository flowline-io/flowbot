package okr

import (
	"fmt"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/form"
)

const (
	CreateObjectiveFormID      = "create_objective"
	UpdateObjectiveFormID      = "update_objective"
	CreateKeyResultFormID      = "create_key_result"
	UpdateKeyResultFormID      = "Update_key_result"
	CreateKeyResultValueFormID = "create_key_result_value"
	CreateTodoFormID           = "create_todo"
	UpdateTodoFormID           = "update_todo"
)

var formRules = []form.Rule{
	{
		Id: CreateObjectiveFormID,
		Handler: func(ctx types.Context, values types.KV) types.MsgPayload {
			var objective model.Objective
			for key, value := range values {
				switch key {
				case "title":
					objective.Title = value.(string)
				case "memo":
					objective.Memo = value.(string)
				case "motive":
					objective.Motive = value.(string)
				case "feasibility":
					objective.Feasibility = value.(string)
				}
			}

			objective.UID = ctx.AsUser.String()
			objective.Topic = ctx.Topic
			_, err := store.Database.CreateObjective(&objective)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: fmt.Sprintf("failed, form [%s]", ctx.FormId)}
			}

			return types.TextMsg{Text: fmt.Sprintf("ok, form [%s]", ctx.FormId)}
		},
	},
	{
		Id: UpdateObjectiveFormID,
		Handler: func(ctx types.Context, values types.KV) types.MsgPayload {
			var objective model.Objective
			for key, value := range values {
				switch key {
				case "sequence":
					objective.Sequence = value.(int32)
				case "title":
					objective.Title = value.(string)
				case "memo":
					objective.Memo = value.(string)
				case "motive":
					objective.Motive = value.(string)
				case "feasibility":
					objective.Feasibility = value.(string)
				}
			}

			objective.UID = ctx.AsUser.String()
			objective.Topic = ctx.Topic
			err := store.Database.UpdateObjective(&objective)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: fmt.Sprintf("failed, form [%s]", ctx.FormId)}
			}

			return types.TextMsg{Text: fmt.Sprintf("ok, form [%s]", ctx.FormId)}
		},
	},
	{
		Id: CreateKeyResultFormID,
		Handler: func(ctx types.Context, values types.KV) types.MsgPayload {
			objectiveSequence := int64(0)
			var keyResult model.KeyResult
			for key, value := range values {
				switch key {
				case "objective_sequence":
					objectiveSequence = value.(int64)
				case "title":
					keyResult.Title = value.(string)
				case "memo":
					keyResult.Memo = value.(string)
				case "initial_value":
					keyResult.InitialValue = int32(value.(int64))
				case "target_value":
					keyResult.TargetValue = int32(value.(int64))
				case "value_mode":
					keyResult.ValueMode = model.ValueModeType(value.(string))
				}
			}

			objective, err := store.Database.GetObjectiveBySequence(ctx.AsUser, ctx.Topic, objectiveSequence)
			if err != nil {
				return nil
			}

			// check
			if keyResult.TargetValue <= 0 {
				return nil
			}
			if keyResult.ValueMode != model.ValueSumMode &&
				keyResult.ValueMode != model.ValueLastMode &&
				keyResult.ValueMode != model.ValueAvgMode &&
				keyResult.ValueMode != model.ValueMaxMode {
				return nil
			}

			// store
			if keyResult.InitialValue > 0 {
				keyResult.CurrentValue = keyResult.InitialValue
			}
			keyResult.ObjectiveID = objective.ID
			keyResult.UID = ctx.AsUser.String()
			keyResult.Topic = ctx.Topic
			_, err = store.Database.CreateKeyResult(&keyResult)
			if err != nil {
				return nil
			}

			// aggregate
			err = store.Database.AggregateObjectiveValue(objective.ID)
			if err != nil {
				return nil
			}

			return types.TextMsg{Text: fmt.Sprintf("ok, form [%s]", ctx.FormId)}
		},
	},
	{
		Id: UpdateKeyResultFormID,
		Handler: func(ctx types.Context, values types.KV) types.MsgPayload {
			var keyResult model.KeyResult
			for key, value := range values {
				switch key {
				case "sequence":
					keyResult.Sequence = value.(int32)
				case "title":
					keyResult.Title = value.(string)
				case "memo":
					keyResult.Memo = value.(string)
				case "target_value":
					keyResult.TargetValue = int32(value.(int64))
				case "value_mode":
					keyResult.ValueMode = model.ValueModeType(value.(string))
				}
			}

			// check
			if keyResult.TargetValue <= 0 {
				return nil
			}
			if keyResult.ValueMode != model.ValueSumMode &&
				keyResult.ValueMode != model.ValueLastMode &&
				keyResult.ValueMode != model.ValueAvgMode &&
				keyResult.ValueMode != model.ValueMaxMode {
				return nil
			}

			keyResult.UID = ctx.AsUser.String()
			keyResult.Topic = ctx.Topic
			err := store.Database.UpdateKeyResult(&keyResult)
			if err != nil {
				return nil
			}

			// update value
			reply, err := store.Database.GetKeyResultBySequence(ctx.AsUser, ctx.Topic, int64(keyResult.Sequence))
			if err != nil {
				return nil
			}
			err = store.Database.AggregateKeyResultValue(reply.ID)
			if err != nil {
				return nil
			}

			return types.TextMsg{Text: fmt.Sprintf("ok, form [%s]", ctx.FormId)}
		},
	},
	{
		Id: CreateKeyResultValueFormID,
		Handler: func(ctx types.Context, values types.KV) types.MsgPayload {
			keyResultSequence := values["key_result_sequence"].(int64)
			value := int32(values["value"].(int64))

			keyResult, err := store.Database.GetKeyResultBySequence(ctx.AsUser, ctx.Topic, keyResultSequence)
			if err != nil {
				return nil
			}
			_, err = store.Database.CreateKeyResultValue(&model.KeyResultValue{Value: value, KeyResultID: keyResult.ID})
			if err != nil {
				return nil
			}
			err = store.Database.AggregateKeyResultValue(keyResult.ID)
			if err != nil {
				return nil
			}
			err = store.Database.AggregateObjectiveValue(keyResult.ObjectiveID)
			if err != nil {
				return nil
			}

			return types.TextMsg{Text: fmt.Sprintf("ok, form [%s]", ctx.FormId)}
		},
	},
	{
		Id: CreateTodoFormID,
		Handler: func(ctx types.Context, values types.KV) types.MsgPayload {
			var todo model.Todo
			for key, value := range values {
				switch key {
				case "content":
					todo.Content = value.(string)
				case "category":
					todo.Category = value.(string)
				case "remark":
					todo.Remark = value.(string)
				case "priority":
					todo.Priority = value.(int32)
				}
			}

			todo.UID = ctx.AsUser.String()
			todo.Topic = ctx.Topic
			_, err := store.Database.CreateTodo(&todo)
			if err != nil {
				flog.Error(err)
				return nil
			}

			return types.TextMsg{Text: fmt.Sprintf("ok, form [%s]", ctx.FormId)}
		},
	},
	{
		Id: UpdateTodoFormID,
		Handler: func(ctx types.Context, values types.KV) types.MsgPayload {
			var todo model.Todo
			for key, value := range values {
				switch key {
				case "sequence":
					todo.Sequence = value.(int32)
				case "content":
					todo.Content = value.(string)
				case "category":
					todo.Category = value.(string)
				case "remark":
					todo.Remark = value.(string)
				case "priority":
					todo.Priority = value.(int32)
				}
			}
			todo.UID = ctx.AsUser.String()
			todo.Topic = ctx.Topic
			err := store.Database.UpdateTodo(&todo)
			if err != nil {
				flog.Error(err)
				return nil
			}

			return types.TextMsg{Text: fmt.Sprintf("ok, form [%s]", ctx.FormId)}
		},
	},
}
