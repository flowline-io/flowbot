package script

import (
	"context"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/cmd/agent/client"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

const (
	// exec script default timeout
	defaultTimeout = time.Hour
)

type Rule struct {
	Id         string `json:"id" river:"unique"`
	When       string
	Path       string
	Timeout    time.Duration
	Version    string
	Desciption string
	Retries    int
	Echo       bool
	Once       bool
}

func (Rule) Kind() string {
	return "script"
}

func (r Rule) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 1 + r.Retries,
		UniqueOpts: river.UniqueOpts{
			ByArgs: true,
			ByState: []rivertype.JobState{
				rivertype.JobStateAvailable,
				rivertype.JobStatePending,
				rivertype.JobStateRetryable,
				rivertype.JobStateRunning,
				rivertype.JobStateScheduled,
			},
		},
	}
}

type ExecScriptWorker struct {
	// An embedded WorkerDefaults sets up default methods to fulfill the rest of
	// the Worker interface:
	river.WorkerDefaults[Rule]
}

func (w *ExecScriptWorker) Work(ctx context.Context, job *river.Job[Rule]) (err error) {
	// once check
	if job.Args.Once {
		f, err := onceLock(job.Args.Id)
		if err != nil {
			return fmt.Errorf("failed to get once lock: %w", err)
		}
		defer func() {
			if err == nil {
				_, writeErr := f.WriteString("1")
				if writeErr != nil {
					flog.Error(fmt.Errorf("failed to write once lock: %w", writeErr))
				}
				_ = f.Close()
			}
		}()
	}

	task, err := execScript(ctx, job.Args)
	if err != nil {
		return fmt.Errorf("failed to execute script: %w", err)
	}
	if task.Error != "" {
		return fmt.Errorf("execute script error: %s", task.Error)
	}
	if task.Result != "" {
		flog.Debug("[script] exec result %v", task.Result)
		if job.Args.Echo {
			err = client.Message(task.Result)
			if err != nil {
				flog.Error(fmt.Errorf("failed to send echo message: %w", err))
			}
		}
	}
	return nil
}

func (w *ExecScriptWorker) Timeout(job *river.Job[Rule]) time.Duration {
	if job.Args.Timeout == 0 {
		return defaultTimeout
	}
	return job.Args.Timeout
}

type ErrorHandler struct{}

func (*ErrorHandler) HandleError(ctx context.Context, job *rivertype.JobRow, err error) *river.ErrorHandlerResult {
	flog.Error(fmt.Errorf("[script] job errored with: %w", err))
	return nil
}

func (*ErrorHandler) HandlePanic(ctx context.Context, job *rivertype.JobRow, panicVal any, trace string) *river.ErrorHandlerResult {
	flog.Error(fmt.Errorf("[script] job panicked with: %v", panicVal))
	flog.Warn("[script] Stack trace: %s\n", trace)

	// Cancel the job to prevent it from being retried:
	return &river.ErrorHandlerResult{
		SetCancelled: true,
	}
}

type LogHook struct {
	river.HookDefaults
}

func (l *LogHook) InsertBegin(ctx context.Context, params *rivertype.JobInsertParams) error {
	flog.Debug("[script] [hook] inserting job with kind %q", params.Kind)
	return nil
}

func (l *LogHook) WorkBegin(ctx context.Context, job *rivertype.JobRow) error {
	flog.Debug("[script] [hook] working job with id %q", job.Kind)
	return nil
}

func (l *LogHook) WorkEnd(ctx context.Context, err error) error {
	flog.Debug("[script] [hook] working job ended with %v", err)
	return nil
}
