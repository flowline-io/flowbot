package workflow

import (
	"context"
	"github.com/flowline-io/flowbot/internal/bots"
	"github.com/flowline-io/flowbot/internal/ruleset/command"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/workflow"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/parser"
	"time"
)

var commandRules = []command.Rule{
	{
		Define: "task run",
		Help:   `Run one task`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return bots.FormMsg(ctx, runOneTaskFormID)
		},
	},
	{
		Define: "task create",
		Help:   `Create one task`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return bots.FormMsg(ctx, createOneTaskFormID)
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
				return types.TextMsg{Text: err.Error()}
			}

			return types.KVMsg{
				"job_id":  step.JobID,
				"node_id": step.NodeID,
				"action":  step.Action,
				"error":   step.Error,
			}
		},
	},
}
