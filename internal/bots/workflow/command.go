package workflow

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/workflow"
	"github.com/flowline-io/flowbot/pkg/chatbot"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/command"
	statsLib "github.com/montanaflynn/stats"
	"gorm.io/gorm"
)

var commandRules = []command.Rule{
	{
		Define: "task run",
		Help:   `Run one task`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return chatbot.FormMsg(ctx, runOneTaskFormID)
		},
	},
	{
		Define: "task create",
		Help:   `Create one task`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return chatbot.FormMsg(ctx, createOneTaskFormID)
		},
	},
	{
		Define: "task run [number]",
		Help:   `Run task`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			workflowId, _ := tokens[2].Value.Int64()

			// get workflow
			wf, err := store.Database.GetWorkflow(workflowId)
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}
			if wf.State == model.WorkflowDisable {
				flog.Debug("workflow %d is disabled", wf.ID)
				return types.TextMsg{Text: "workflow is disabled"}
			}

			triggerId := int64(0)
			for _, trigger := range wf.Triggers {
				if trigger.Type == model.TriggerManual {
					triggerId = trigger.ID
				}
			}
			if triggerId == 0 {
				return types.TextMsg{Text: "no manual trigger"}
			}

			// update trigger count
			err = store.Database.IncreaseWorkflowTriggerCount(triggerId, 1)
			if err != nil {
				flog.Error(err)
			}

			// create job
			dagId := int64(0)
			scriptVersion := int32(0)
			if len(wf.Dag) > 0 {
				dagId = wf.Dag[0].ID
				scriptVersion = wf.Dag[0].ScriptVersion
			}
			now := time.Now()
			job := &model.Job{
				UID:           wf.UID,
				Topic:         wf.Topic,
				WorkflowID:    wf.ID,
				DagID:         dagId,
				TriggerID:     triggerId,
				ScriptVersion: scriptVersion,
				State:         model.JobReady,
				StartedAt:     &now,
			}
			_, err = store.Database.CreateJob(job)
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}
			err = workflow.SyncJob(context.Background(), job)
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			return types.TextMsg{Text: "ok"}
		},
	},
	{
		Define: "task error",
		Help:   `get workflow step's last error message`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			step, err := store.Database.GetLastStepByState(model.StepFailed)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return types.TextMsg{Text: "no failed workflow step"}
				}
				return types.TextMsg{Text: fmt.Sprintf("failed to get last step: %s", err.Error())}
			}

			return types.KVMsg{
				"job_id":  step.JobID,
				"node_id": step.NodeID,
				"action":  step.Action,
				"error":   step.Error,
			}
		},
	},
	{
		Define: "task start [number]",
		Help:   `Start task`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			workflowId, _ := tokens[2].Value.Int64()

			err := store.Database.UpdateWorkflowState(workflowId, model.WorkflowEnable)
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}
			err = store.Database.UpdateWorkflowTriggerStateByWorkflowId(workflowId, model.WorkflowTriggerEnable)
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			return types.TextMsg{Text: "ok"}
		},
	},
	{
		Define: "task stop [number]",
		Help:   `Stop task`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			workflowId, _ := tokens[2].Value.Int64()

			err := store.Database.UpdateWorkflowState(workflowId, model.WorkflowDisable)
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}
			err = store.Database.UpdateWorkflowTriggerStateByWorkflowId(workflowId, model.WorkflowTriggerDisable)
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			return types.TextMsg{Text: "ok"}
		},
	},
	{
		Define: "workflow stat",
		Help:   `workflow job statisticians`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			jobs, err := store.Database.GetJobsByState(model.JobSucceeded)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: err.Error()}
			}
			steps, err := store.Database.GetStepsByState(model.StepSucceeded)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: err.Error()}
			}

			jobElapsed := make([]float64, 0, len(jobs))
			for _, job := range jobs {
				if job.StartedAt == nil || job.EndedAt == nil {
					continue
				}
				elapsed := job.EndedAt.Sub(*job.StartedAt).Seconds()
				if elapsed < 0 {
					continue
				}
				jobElapsed = append(jobElapsed, elapsed)
			}

			stepElapsed := make([]float64, 0, len(steps))
			for _, step := range steps {
				if step.StartedAt == nil || step.EndedAt == nil {
					continue
				}
				elapsed := step.EndedAt.Sub(*step.StartedAt).Seconds()
				if elapsed < 0 {
					continue
				}
				stepElapsed = append(stepElapsed, elapsed)
			}

			str := strings.Builder{}
			minVal, _ := statsLib.Min(jobElapsed)
			medianVal, _ := statsLib.Median(jobElapsed)
			maxVal, _ := statsLib.Max(jobElapsed)
			avgVal, _ := statsLib.Mean(jobElapsed)
			varVal, _ := statsLib.Variance(jobElapsed)
			_, _ = str.WriteString(fmt.Sprintf("Jobs total %d, min: %f, median: %f, max: %f, avg: %f, variance: %f \n",
				len(jobElapsed), minVal, medianVal, maxVal, avgVal, varVal))

			minVal, _ = statsLib.Min(stepElapsed)
			medianVal, _ = statsLib.Median(stepElapsed)
			maxVal, _ = statsLib.Max(stepElapsed)
			avgVal, _ = statsLib.Mean(stepElapsed)
			varVal, _ = statsLib.Variance(stepElapsed)
			_, _ = str.WriteString(fmt.Sprintf("Steps total %d, min: %f, median: %f, max: %f, avg: %f, variance: %f \n",
				len(stepElapsed), minVal, medianVal, maxVal, avgVal, varVal))

			return types.TextMsg{Text: str.String()}
		},
	},
	{
		Define: "workflow queue",
		Help:   `workflow queue statisticians`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			inspector := workflow.GetInspector()
			queues, err := inspector.Queues()
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			str := strings.Builder{}
			for _, queueName := range queues {
				info, err := inspector.GetQueueInfo(queueName)
				if err != nil {
					return types.TextMsg{Text: err.Error()}
				}

				_, _ = str.WriteString(fmt.Sprintf("queue %s: size %d memory %v processed %d failed %d \n",
					info.Queue, info.Size, humanize.Bytes(uint64(info.MemoryUsage)), info.Processed, info.Failed))
			}

			return types.TextMsg{Text: str.String()}
		},
	},
	{
		Define: "workflow history",
		Help:   `workflow task history`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			inspector := workflow.GetInspector()
			queues, err := inspector.Queues()
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			str := strings.Builder{}
			for _, queueName := range queues {
				stats, err := inspector.History(queueName, 7)
				if err != nil {
					return types.TextMsg{Text: err.Error()}
				}
				_, _ = str.WriteString(fmt.Sprintf("queue %s:", queueName))
				for _, info := range stats {
					_, _ = str.WriteString(fmt.Sprintf("%s -> processed %d failed %d, ",
						info.Date.Format(time.DateOnly), info.Processed, info.Failed))
				}
				_, _ = str.WriteString("\n")
			}

			return types.TextMsg{Text: str.String()}
		},
	},
	{
		Define: "workflow list",
		Help:   `print workflow list`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			list, err := store.Database.ListWorkflows(ctx.AsUser, ctx.Topic)
			if err != nil {
				return types.TextMsg{Text: err.Error()}
			}

			total := len(list)

			// filter state enabled
			for i, item := range list {
				if item.State != model.WorkflowEnable {
					list = append(list[:i], list[i+1:]...)
					total--
				}
			}

			return types.InfoMsg{
				Title: fmt.Sprintf("workflows %v", total),
				Model: list,
			}
		},
	},
}
