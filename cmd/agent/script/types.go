package script

import (
	"context"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

type Rule struct {
	Id      string `json:"id" river:"unique"`
	When    string
	Path    string
	Timeout time.Duration
}

func (Rule) Kind() string {
	return "script"
}

func (Rule) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 1,
		// UniqueOpts: river.UniqueOpts{
		// 	ByArgs: true,
		// 	ByState: []rivertype.JobState{
		// 		rivertype.JobStateAvailable,
		// 		rivertype.JobStatePending,
		// 		rivertype.JobStateRetryable,
		// 		rivertype.JobStateRunning,
		// 		rivertype.JobStateScheduled,
		// 	},
		// },
	}
}

type ExecScriptWorker struct {
	// An embedded WorkerDefaults sets up default methods to fulfill the rest of
	// the Worker interface:
	river.WorkerDefaults[Rule]
}

func (w *ExecScriptWorker) Work(ctx context.Context, job *river.Job[Rule]) (err error) {
	return execScript(ctx, job.Args)
}

func (w *ExecScriptWorker) Timeout(job *river.Job[Rule]) time.Duration {
	if job.Args.Timeout == 0 {
		return time.Hour
	}
	return job.Args.Timeout
}

type ErrorHandler struct{}

func (*ErrorHandler) HandleError(ctx context.Context, job *rivertype.JobRow, err error) *river.ErrorHandlerResult {
	flog.Error(fmt.Errorf("Job errored with: %s", err))
	return nil
}

func (*ErrorHandler) HandlePanic(ctx context.Context, job *rivertype.JobRow, panicVal any, trace string) *river.ErrorHandlerResult {
	flog.Error(fmt.Errorf("Job panicked with: %v", panicVal))
	flog.Warn("Stack trace: %s\n", trace)

	// Cancel the job to prevent it from being retried:
	return &river.ErrorHandlerResult{
		SetCancelled: true,
	}
}
