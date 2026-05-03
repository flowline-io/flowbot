package server

import (
	"context"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/mysql"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"go.uber.org/fx"
)

func newDatabaseAdapter(lc fx.Lifecycle, _ *config.Type) (store.Adapter, error) {
	// init database
	mysql.Init()
	store.Init()

	// Open database
	err := store.Store.Open(config.App.Store)
	if err != nil {
		return nil, fmt.Errorf("failed to open DB, %w", err)
	}

	// migrate
	if err := store.Migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate DB, %w", err)
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return nil
		},
		OnStop: func(ctx context.Context) error {
			ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			done := make(chan struct{})
			go func() {
				defer close(done)
				if err := store.Store.Close(); err != nil {
					flog.Error(fmt.Errorf("database close error: %w", err))
				}
			}()

			select {
			case <-done:
				flog.Info("database closed")
			case <-ctx.Done():
				flog.Error(fmt.Errorf("database close timed out: %w", ctx.Err()))
			}
			return nil
		},
	})

	return store.Database, nil
}
