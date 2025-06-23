package script

import (
	"context"
	"database/sql"
	"errors"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riversqlite"
	_ "modernc.org/sqlite" //revive:disable
)

func (e *Engine) queue() {
	dbPool, err := sql.Open("sqlite", "file:./river.sqlite3?_pragma=journal_mode(WAL)&_txlock=immediate")
	if err != nil {
		flog.Error(err)
		return
	}
	defer dbPool.Close()

	dbPool.SetMaxOpenConns(1)

	workers := river.NewWorkers()
	river.AddWorker(workers, &ExecScriptWorker{})

	riverClient, err := river.NewClient(riversqlite.New(dbPool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 100},
		},
		Workers: workers,
	})
	if err != nil {
		flog.Error(err)
		return
	}
	e.client = riverClient

	// Run the client inline. All executed jobs will inherit from ctx:
	if err := riverClient.Start(context.Background()); err != nil {
		flog.Error(err)
	}

	<-e.stop
	flog.Info("stop queue client")
}

func (e *Engine) pushQueue(ctx context.Context, r Rule) error {
	if e.client == nil {
		return errors.New("queue client is nil")
	}
	result, err := e.client.Insert(ctx, r, nil)
	if err != nil {
		return err
	}
	flog.Info("push exec script job %+v", result.Job.ID)
	return nil
}
