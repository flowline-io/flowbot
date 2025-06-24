package script

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"

	"github.com/adrg/xdg"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riversqlite"
	"github.com/riverqueue/river/rivermigrate"
	"github.com/riverqueue/river/rivertype"
	_ "modernc.org/sqlite" //revive:disable
)

func (e *Engine) queue() {
	if xdg.ConfigHome == "" {
		flog.Error(errors.New("xdg.ConfigHome is empty"))
		return
	}
	agentConfigPath := fmt.Sprintf("%s/flowbot", xdg.ConfigHome)
	if err := os.MkdirAll(agentConfigPath, 0600); err != nil {
		flog.Error(err)
		return
	}

	flog.Info("queue database path: %s/river.sqlite3", agentConfigPath)
	dbPool, err := sql.Open("sqlite", fmt.Sprintf("file:%s/river.sqlite3?_pragma=journal_mode(WAL)&_txlock=immediate", agentConfigPath))
	if err != nil {
		flog.Error(err)
		return
	}
	dbPool.SetMaxOpenConns(1)
	defer dbPool.Close()

	workers := river.NewWorkers()
	river.AddWorker(workers, &ExecScriptWorker{})

	riverClient, err := river.NewClient(riversqlite.New(dbPool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 100},
		},
		Workers:      workers,
		ErrorHandler: &ErrorHandler{},
		Hooks: []rivertype.Hook{
			&LogHook{},
		},
	})
	if err != nil {
		flog.Error(err)
		return
	}
	e.client = riverClient

	// migrate
	migrator, err := rivermigrate.New(riversqlite.New(dbPool), &rivermigrate.Config{})
	if err != nil {
		flog.Error(err)
		return
	}
	res, err := migrator.Migrate(context.Background(), rivermigrate.DirectionUp, nil)
	if err != nil {
		flog.Error(err)
		return
	}
	for _, migrateVersion := range res.Versions {
		flog.Info("migrate %s -> %d:%s in %s", res.Direction, migrateVersion.Version, migrateVersion.Name, migrateVersion.Duration)
	}

	// Run the client inline. All executed jobs will inherit from ctx:
	if err := riverClient.Start(context.Background()); err != nil {
		flog.Error(err)
	}
	e.queueStarted <- struct{}{}

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
