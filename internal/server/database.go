package server

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/mysql"
	"github.com/flowline-io/flowbot/pkg/config"
	"go.uber.org/fx"
)

func newDatabaseAdapter(lc fx.Lifecycle, _ config.Type) (store.Adapter, error) {
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
			return store.Store.Close()
		},
	})

	return store.Database, nil
}
