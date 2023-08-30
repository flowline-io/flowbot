package okr

import (
	"fmt"
	"github.com/flowline-io/flowbot/internal/ruleset/form"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
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

			objective.UID = ctx.AsUser.UserId()
			objective.Topic = ctx.Original
			_, err := store.Chatbot.CreateObjective(&objective)
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

			objective.UID = ctx.AsUser.UserId()
			objective.Topic = ctx.Original
			err := store.Chatbot.UpdateObjective(&objective)
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

			objective, err := store.Chatbot.GetObjectiveBySequence(ctx.AsUser, ctx.Original, objectiveSequence)
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
			keyResult.UID = ctx.AsUser.UserId()
			keyResult.Topic = ctx.Original
			_, err = store.Chatbot.CreateKeyResult(&keyResult)
			if err != nil {
				return nil
			}

			// aggregate
			err = store.Chatbot.AggregateObjectiveValue(int64(objective.ID))
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

			keyResult.UID = ctx.AsUser.UserId()
			keyResult.Topic = ctx.Original
			err := store.Chatbot.UpdateKeyResult(&keyResult)
			if err != nil {
				return nil
			}

			// update value
			reply, err := store.Chatbot.GetKeyResultBySequence(ctx.AsUser, ctx.Original, int64(keyResult.Sequence))
			if err != nil {
				return nil
			}
			err = store.Chatbot.AggregateKeyResultValue(int64(reply.ID))
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

			keyResult, err := store.Chatbot.GetKeyResultBySequence(ctx.AsUser, ctx.Original, keyResultSequence)
			if err != nil {
				return nil
			}
			_, err = store.Chatbot.CreateKeyResultValue(&model.KeyResultValue{Value: value, KeyResultID: keyResult.ID})
			if err != nil {
				return nil
			}
			err = store.Chatbot.AggregateKeyResultValue(int64(keyResult.ID))
			if err != nil {
				return nil
			}
			err = store.Chatbot.AggregateObjectiveValue(int64(keyResult.ObjectiveID))
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

			todo.UID = ctx.AsUser.UserId()
			todo.Topic = ctx.Original
			_, err := store.Chatbot.CreateTodo(&todo)
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
			todo.UID = ctx.AsUser.UserId()
			todo.Topic = ctx.Original
			err := store.Chatbot.UpdateTodo(&todo)
			if err != nil {
				flog.Error(err)
				return nil
			}

			return types.TextMsg{Text: fmt.Sprintf("ok, form [%s]", ctx.FormId)}
		},
	},
}
