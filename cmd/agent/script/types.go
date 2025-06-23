package script

import (
	"context"
	"time"

	"github.com/riverqueue/river"
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

func (w *ExecScriptWorker) Work(ctx context.Context, job *river.Job[Rule]) error {
	return execScript(job.Args)
}
