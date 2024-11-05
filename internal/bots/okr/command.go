package okr

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/ruleset/command"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/parser"
	"gorm.io/gorm"
)

var commandRules = []command.Rule{
	{
		Define: `obj list`,
		Help:   `List objectives`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			items, err := store.Database.ListObjectives(ctx.AsUser, ctx.Topic)
			if err != nil {
				flog.Error(err)
				return nil
			}

			var header []string
			var row [][]interface{}
			if len(items) > 0 {
				header = []string{"Sequence", "Title", "Current Value", "Total Value"}
				for _, v := range items {
					row = append(row, []interface{}{strconv.Itoa(int(v.Sequence)), v.Title, strconv.Itoa(int(v.CurrentValue)), strconv.Itoa(int(v.TotalValue))})
				}
			}
			if len(row) == 0 {
				return types.TextMsg{Text: "Empty"}
			}

			return bots.StorePage(ctx, model.PageTable, "Objectives", types.TableMsg{Title: "Objectives", Header: header, Row: row})
		},
	},
	{
		Define: `obj [number]`,
		Help:   `View objective`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			param := types.KV{}
			sequence, _ := tokens[1].Value.Int64()
			param["sequence"] = sequence

			url, err := bots.PageURL(ctx, okrPageId, param, 24*time.Hour)
			if err != nil {
				return types.TextMsg{Text: "error"}
			}

			return types.LinkMsg{Url: url}
		},
	},
	{
		Define: `obj del [number]`,
		Help:   `Delete objective`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			sequence, _ := tokens[2].Value.Int64()

			err := store.Database.DeleteObjectiveBySequence(ctx.AsUser, ctx.Topic, sequence)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: "failed"}
			}

			return types.TextMsg{Text: "ok"}
		},
	},
	{
		Define: `obj update [number]`,
		Help:   `Update objective`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			sequence, _ := tokens[2].Value.Int64()

			item, err := store.Database.GetObjectiveBySequence(ctx.AsUser, ctx.Topic, sequence)
			if err != nil {
				flog.Error(err)
				return nil
			}

			return bots.StoreForm(ctx, types.FormMsg{
				ID:    UpdateObjectiveFormID,
				Title: fmt.Sprintf("Update Objective #%d", sequence),
				Field: []types.FormField{
					{
						Key:       "sequence",
						Type:      types.FormFieldNumber,
						ValueType: types.FormFieldValueInt64,
						Value:     item.Sequence,
						Label:     "Sequence",
					},
					{
						Key:       "title",
						Type:      types.FormFieldText,
						ValueType: types.FormFieldValueString,
						Value:     item.Title,
						Label:     "Title",
					},
					{
						Key:       "memo",
						Type:      types.FormFieldText,
						ValueType: types.FormFieldValueString,
						Value:     item.Memo,
						Label:     "Memo",
					},
					{
						Key:       "motive",
						Type:      types.FormFieldText,
						ValueType: types.FormFieldValueString,
						Value:     item.Motive,
						Label:     "Motive",
					},
					{
						Key:       "feasibility",
						Type:      types.FormFieldText,
						ValueType: types.FormFieldValueString,
						Value:     item.Feasibility,
						Label:     "Feasibility",
					},
				},
			})
		},
	},
	{
		Define: `obj create`,
		Help:   `Create Objective`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return bots.StoreForm(ctx, types.FormMsg{
				ID:    CreateObjectiveFormID,
				Title: "Create Objective",
				Field: []types.FormField{
					{
						Key:       "title",
						Type:      types.FormFieldText,
						ValueType: types.FormFieldValueString,
						Label:     "Title",
					},
					{
						Key:       "memo",
						Type:      types.FormFieldText,
						ValueType: types.FormFieldValueString,
						Label:     "Memo",
					},
					{
						Key:       "motive",
						Type:      types.FormFieldText,
						ValueType: types.FormFieldValueString,
						Label:     "Motive",
					},
					{
						Key:       "feasibility",
						Type:      types.FormFieldText,
						ValueType: types.FormFieldValueString,
						Label:     "Feasibility",
					},
					{
						Key:       "is_plan",
						Type:      types.FormFieldRadio,
						ValueType: types.FormFieldValueBool,
						Label:     "IsPlan",
						Option:    []string{"true", "false"},
					},
					{
						Key:       "plan_start",
						Type:      types.FormFieldText,
						ValueType: types.FormFieldValueString,
						Label:     "PlanStart",
					},
					{
						Key:       "plan_end",
						Type:      types.FormFieldText,
						ValueType: types.FormFieldValueString,
						Label:     "PlanEnd",
					},
				},
			})
		},
	},
	{
		Define: `kr list`,
		Help:   `List KeyResult`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			items, err := store.Database.ListKeyResults(ctx.AsUser, ctx.Topic)
			if err != nil {
				flog.Error(err)
				return nil
			}

			var header []string
			var row [][]interface{}
			if len(items) > 0 {
				header = []string{"Sequence", "Title", "Current Value", "Target Value"}
				for _, v := range items {
					row = append(row, []interface{}{strconv.Itoa(int(v.Sequence)), v.Title, strconv.Itoa(int(v.CurrentValue)), strconv.Itoa(int(v.TargetValue))})
				}
			}

			return bots.StorePage(ctx, model.PageTable, "KeyResults", types.TableMsg{
				Title:  "KeyResults",
				Header: header,
				Row:    row,
			})
		},
	},
	{
		Define: `kr create`,
		Help:   `Create KeyResult`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return bots.StoreForm(ctx, types.FormMsg{
				ID:    CreateKeyResultFormID,
				Title: "Create Key Result",
				Field: []types.FormField{
					{
						Key:       "objective_sequence",
						Type:      types.FormFieldNumber,
						ValueType: types.FormFieldValueInt64,
						Label:     "Objective Sequence",
					},
					{
						Key:       "title",
						Type:      types.FormFieldText,
						ValueType: types.FormFieldValueString,
						Label:     "Title",
					},
					{
						Key:       "memo",
						Type:      types.FormFieldText,
						ValueType: types.FormFieldValueString,
						Label:     "Memo",
					},
					{
						Key:       "initial_value",
						Type:      types.FormFieldNumber,
						ValueType: types.FormFieldValueInt64,
						Label:     "initial value",
					},
					{
						Key:       "target_value",
						Type:      types.FormFieldNumber,
						ValueType: types.FormFieldValueInt64,
						Label:     "target value",
					},
					{
						Key:       "value_mode",
						Type:      types.FormFieldSelect,
						ValueType: types.FormFieldValueString,
						Label:     "value mode",
						Option:    []string{"avg", "max", "sum", "last"},
					},
				},
			})
		},
	},
	{
		Define: `kr [number]`,
		Help:   `View KeyResult`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			sequence, _ := tokens[1].Value.Int64()

			item, err := store.Database.GetKeyResultBySequence(ctx.AsUser, ctx.Topic, sequence)
			if err != nil {
				flog.Error(err)
				return nil
			}

			return types.InfoMsg{
				Title: fmt.Sprintf("KeyResult #%d", sequence),
				Model: item,
			}
		},
	},
	{
		Define: `kr del [number]`,
		Help:   `Delete KeyResult`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			sequence, _ := tokens[2].Value.Int64()

			err := store.Database.DeleteKeyResultBySequence(ctx.AsUser, ctx.Topic, sequence)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: "failed"}
			}

			return types.TextMsg{Text: "ok"}
		},
	},
	{
		Define: `kr update [number]`,
		Help:   `Update KeyResult`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			sequence, _ := tokens[2].Value.Int64()

			item, err := store.Database.GetKeyResultBySequence(ctx.AsUser, ctx.Topic, sequence)
			if err != nil {
				flog.Error(err)
				return nil
			}

			return bots.StoreForm(ctx, types.FormMsg{
				ID:    UpdateKeyResultFormID,
				Title: fmt.Sprintf("Update KeyResult #%d", sequence),
				Field: []types.FormField{
					{
						Key:       "sequence",
						Type:      types.FormFieldNumber,
						ValueType: types.FormFieldValueInt64,
						Label:     "Sequence",
						Value:     item.Sequence,
					},
					{
						Key:       "title",
						Type:      types.FormFieldText,
						ValueType: types.FormFieldValueString,
						Label:     "Title",
						Value:     item.Title,
					},
					{
						Key:       "memo",
						Type:      types.FormFieldText,
						ValueType: types.FormFieldValueString,
						Label:     "Memo",
						Value:     item.Memo,
					},
					{
						Key:       "target_value",
						Type:      types.FormFieldNumber,
						ValueType: types.FormFieldValueInt64,
						Label:     "target value",
						Value:     item.TargetValue,
					},
					{
						Key:       "value_mode",
						Type:      types.FormFieldSelect,
						ValueType: types.FormFieldValueString,
						Label:     "value mode",
						Option:    []string{"avg", "max", "sum", "last"},
						Value:     item.ValueMode,
					},
				},
			})
		},
	},
	{
		Define: `kr value`,
		Help:   `Create KeyResult value`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return bots.StoreForm(ctx, types.FormMsg{
				ID:    CreateKeyResultValueFormID,
				Title: "Create Key Result value",
				Field: []types.FormField{
					{
						Key:       "key_result_sequence",
						Type:      types.FormFieldNumber,
						ValueType: types.FormFieldValueInt64,
						Label:     "Key Result Sequence",
					},
					{
						Key:       "value",
						Type:      types.FormFieldNumber,
						ValueType: types.FormFieldValueInt64,
						Label:     "Value",
					},
				},
			})
		},
	},
	{
		Define: `kr value [number]`,
		Help:   `List KeyResult value`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			sequence, _ := tokens[2].Value.Int64()

			keyResult, err := store.Database.GetKeyResultBySequence(ctx.AsUser, ctx.Topic, sequence)
			if err != nil {
				flog.Error(err)
				return nil
			}

			items, err := store.Database.GetKeyResultValues(keyResult.ID)
			if err != nil {
				flog.Error(err)
				return nil
			}

			var header []string
			var row [][]interface{}
			if len(items) > 0 {
				header = []string{"Value", "Datetime"}
				for _, v := range items {
					row = append(row, []interface{}{strconv.Itoa(int(v.Value)), v.CreatedAt})
				}
			}

			title := fmt.Sprintf("KeyResult #%d Values", sequence)
			return bots.StorePage(ctx, model.PageTable, title, types.TableMsg{
				Title:  title,
				Header: header,
				Row:    row,
			})
		},
	},
	{
		Define: `todo list`,
		Help:   `List todo`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			items, err := store.Database.ListTodos(ctx.AsUser, ctx.Topic)
			if err != nil {
				flog.Error(err)
				return nil
			}

			return types.InfoMsg{
				Title: "Todo",
				Model: items,
			}
		},
	},
	{
		Define: `todo create`,
		Help:   "Create Todo something",
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return bots.StoreForm(ctx, types.FormMsg{
				ID:    CreateTodoFormID,
				Title: "Create todo",
				Field: []types.FormField{
					{
						Key:       "content",
						Type:      types.FormFieldText,
						ValueType: types.FormFieldValueString,
						Label:     "Content",
					},
					{
						Key:       "category",
						Type:      types.FormFieldText,
						ValueType: types.FormFieldValueString,
						Label:     "Category",
					},
					{
						Key:       "remark",
						Type:      types.FormFieldText,
						ValueType: types.FormFieldValueString,
						Label:     "Remark",
					},
					{
						Key:       "priority",
						Type:      types.FormFieldNumber,
						ValueType: types.FormFieldValueInt64,
						Label:     "Priority",
					},
				},
			})
		},
	},
	{
		Define: `todo update [number]`,
		Help:   "Update Todo something",
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			sequence, _ := tokens[2].Value.Int64()

			item, err := store.Database.GetTodoBySequence(ctx.AsUser, ctx.Topic, sequence)
			if err != nil {
				flog.Error(err)
				return nil
			}

			return bots.StoreForm(ctx, types.FormMsg{
				ID:    UpdateTodoFormID,
				Title: fmt.Sprintf("Update Todo #%d", sequence),
				Field: []types.FormField{
					{
						Key:       "sequence",
						Type:      types.FormFieldNumber,
						ValueType: types.FormFieldValueInt64,
						Label:     "Sequence",
						Value:     item.Sequence,
					},
					{
						Key:       "content",
						Type:      types.FormFieldText,
						ValueType: types.FormFieldValueString,
						Label:     "Content",
						Value:     item.Content,
					},
					{
						Key:       "category",
						Type:      types.FormFieldText,
						ValueType: types.FormFieldValueString,
						Label:     "Category",
						Value:     item.Category,
					},
					{
						Key:       "remark",
						Type:      types.FormFieldText,
						ValueType: types.FormFieldValueString,
						Label:     "Remark",
						Value:     item.Remark,
					},
					{
						Key:       "priority",
						Type:      types.FormFieldNumber,
						ValueType: types.FormFieldValueInt64,
						Label:     "Priority",
						Value:     item.Priority,
					},
				},
			})
		},
	},
	{
		Define: `todo complete [number]`,
		Help:   "Complete Todo",
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			sequence, _ := tokens[2].Value.Int64()
			err := store.Database.CompleteTodoBySequence(ctx.AsUser, ctx.Topic, sequence)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: "failed"}
			}

			return types.TextMsg{Text: "ok"}
		},
	},
	{
		Define: `counters`,
		Help:   `List Counter`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			items, err := store.Database.ListCounter(ctx.AsUser, ctx.Topic)
			if err != nil {
				return nil
			}

			var header []string
			var row [][]interface{}
			if len(items) > 0 {
				header = []string{"No", "Title", "Digit"}
				for i, item := range items {
					row = append(row, []interface{}{i + 1, item.Flag, item.Digit})
				}
			}

			return bots.StorePage(ctx, model.PageTable, "Counters", types.TableMsg{
				Title:  "Counters",
				Header: header,
				Row:    row,
			})
		},
	},
	{
		Define: "counter [string]",
		Help:   `Count things`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			flag, _ := tokens[1].Value.String()

			item, err := store.Database.GetCounterByFlag(ctx.AsUser, ctx.Topic, flag)
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}

			if item.ID > 0 {
				return types.KVMsg{
					"Title": item.Flag,
					"Digit": int(item.Digit),
				}
			}

			_, err = store.Database.CreateCounter(&model.Counter{
				UID:    ctx.AsUser.String(),
				Topic:  ctx.Topic,
				Flag:   flag,
				Digit:  1,
				Status: 0,
			})
			if err != nil {
				return nil
			}

			return types.KVMsg{
				"Title": flag,
				"Digit": 1,
			}
		},
	},
	{
		Define: "increase [string]",
		Help:   `Increase Counter`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			flag, _ := tokens[1].Value.String()

			item, err := store.Database.GetCounterByFlag(ctx.AsUser, ctx.Topic, flag)
			if err != nil {
				return nil
			}

			err = store.Database.IncreaseCounter(item.ID, 1)
			if err != nil {
				return nil
			}

			return types.TextMsg{Text: "ok"}
		},
	},
	{
		Define: "decrease [string]",
		Help:   `Decrease Counter`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			flag, _ := tokens[1].Value.String()

			item, err := store.Database.GetCounterByFlag(ctx.AsUser, ctx.Topic, flag)
			if err != nil {
				return nil
			}

			err = store.Database.DecreaseCounter(item.ID, 1)
			if err != nil {
				return nil
			}

			return types.TextMsg{Text: "ok"}
		},
	},
	{
		Define: "reset [string]",
		Help:   `Reset Counter`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			flag, _ := tokens[1].Value.String()

			item, err := store.Database.GetCounterByFlag(ctx.AsUser, ctx.Topic, flag)
			if err != nil {
				return nil
			}

			err = store.Database.IncreaseCounter(item.ID, 1-item.Digit)
			if err != nil {
				return nil
			}

			return types.TextMsg{Text: "ok"}
		},
	},
}
