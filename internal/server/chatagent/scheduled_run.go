package chatagent

import (
	"context"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/cronutil"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

var scheduledRunService = NewService()

// ExecuteScheduledTaskForTest runs one scheduled task; exposed for integration specs.
func ExecuteScheduledTaskForTest(ctx context.Context, task *gen.ChatScheduledTask) {
	executeScheduledTask(ctx, task)
}

// executeScheduledTask runs one scheduled task in an isolated session.
func executeScheduledTask(ctx context.Context, task *gen.ChatScheduledTask) {
	if task == nil || store.Database == nil {
		return
	}
	runSessionID, runFlag, sessionOK := beginScheduledTaskRun(ctx, task)
	if !sessionOK {
		return
	}
	defer closeScheduledTaskSession(ctx, task.Flag, runSessionID)

	reply, runErr := runScheduledTaskPrompt(ctx, runSessionID, task)
	finished := time.Now().UTC()
	persistScheduledTaskRun(ctx, task, runFlag, reply, runErr, finished)
	updateScheduledTaskAfterRun(ctx, task, finished)
	deliverScheduledTaskReply(ctx, task, reply)
	finalizeScheduledTask(ctx, task, runErr)
}

func beginScheduledTaskRun(ctx context.Context, task *gen.ChatScheduledTask) (runSessionID, runFlag string, ok bool) {
	uid := types.Uid(task.UID)
	runSessionID = types.Id()
	runFlag = types.Id()
	if err := CreateSession(ctx, uid, runSessionID); err != nil {
		flog.Error(fmt.Errorf("[chat-agent] scheduled task session create task=%s: %w", task.Flag, err))
		return "", "", false
	}
	if err := store.Database.CreateChatScheduledTaskRun(ctx, &gen.ChatScheduledTaskRun{
		Flag:         runFlag,
		TaskID:       task.Flag,
		RunSessionID: runSessionID,
		State:        string(schema.ChatScheduledTaskRunStateRunning),
		StartedAt:    time.Now().UTC(),
	}); err != nil {
		flog.Error(fmt.Errorf("[chat-agent] scheduled task run record task=%s: %w", task.Flag, err))
		if cerr := CloseSession(ctx, runSessionID); cerr != nil {
			flog.Warn("[chat-agent] scheduled task session close after run record failure task=%s: %v", task.Flag, cerr)
		}
		return "", "", false
	}
	return runSessionID, runFlag, true
}

func closeScheduledTaskSession(ctx context.Context, taskID, runSessionID string) {
	if err := CloseSession(ctx, runSessionID); err != nil {
		flog.Warn("[chat-agent] scheduled task session close task=%s session=%s: %v", taskID, runSessionID, err)
	}
}

func runScheduledTaskPrompt(ctx context.Context, runSessionID string, task *gen.ChatScheduledTask) (string, error) {
	runCtx, cancel := context.WithTimeout(ctx, RunTimeout())
	defer cancel()
	return scheduledRunService.Run(runCtx, RunRequest{
		SessionID: runSessionID,
		Text:      task.Prompt,
	}, nil)
}

func persistScheduledTaskRun(ctx context.Context, task *gen.ChatScheduledTask, runFlag, reply string, runErr error, finished time.Time) {
	runState := string(schema.ChatScheduledTaskRunStateCompleted)
	errText := ""
	if runErr != nil {
		runState = string(schema.ChatScheduledTaskRunStateFailed)
		errText = runErr.Error()
		reply = fmt.Sprintf("Scheduled task failed: %s", runErr.Error())
		flog.Error(fmt.Errorf("[chat-agent] scheduled task run task=%s: %w", task.Flag, runErr))
	}
	if err := store.Database.UpdateChatScheduledTaskRun(ctx, runFlag, store.UpdateChatScheduledTaskRunParams{
		State:      &runState,
		Reply:      &reply,
		Error:      &errText,
		FinishedAt: &finished,
	}); err != nil {
		flog.Warn("[chat-agent] scheduled task run update task=%s: %v", task.Flag, err)
	}
}

func updateScheduledTaskAfterRun(ctx context.Context, task *gen.ChatScheduledTask, finished time.Time) {
	taskUpdate := store.UpdateChatScheduledTaskParams{LastRunAt: &finished}
	if task.ScheduleKind == string(schema.ChatScheduledTaskKindCron) && task.Cron != "" {
		if next, nerr := cronutil.NextRun(task.Cron, finished); nerr == nil {
			taskUpdate.NextRunAt = &next
		} else {
			flog.Warn("[chat-agent] scheduled task next_run_at task=%s: %v", task.Flag, nerr)
		}
	}
	if err := store.Database.UpdateChatScheduledTask(ctx, task.Flag, taskUpdate); err != nil {
		flog.Warn("[chat-agent] scheduled task metadata update task=%s: %v", task.Flag, err)
	}
}

func deliverScheduledTaskReply(ctx context.Context, task *gen.ChatScheduledTask, reply string) {
	if reply != "" {
		deliverScheduledReply(ctx, task, reply)
	}
}

func finalizeScheduledTask(ctx context.Context, task *gen.ChatScheduledTask, runErr error) {
	if task.ScheduleKind != string(schema.ChatScheduledTaskKindOnce) {
		return
	}
	finalState := string(schema.ChatScheduledTaskStateCompleted)
	if runErr != nil {
		finalState = string(schema.ChatScheduledTaskStateFailed)
	}
	if err := store.Database.UpdateChatScheduledTask(ctx, task.Flag, store.UpdateChatScheduledTaskParams{
		State: &finalState,
	}); err != nil {
		flog.Warn("[chat-agent] scheduled task finalize task=%s: %v", task.Flag, err)
	}
	if sched := GlobalScheduler(); sched != nil {
		sched.UnregisterTask(task.Flag)
	}
}
