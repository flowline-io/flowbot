package server

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/fx"

	storepkg "github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/flog"
)

// initPageDataCleanup starts a background goroutine that periodically deletes
// expired page_data rows. It uses fx.Lifecycle hooks for graceful shutdown.
func initPageDataCleanup(lc fx.Lifecycle) {
	ctx, cancel := context.WithCancel(context.Background())

	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			runCleanup()
			go func() {
				ticker := time.NewTicker(1 * time.Hour)
				defer ticker.Stop()
				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						runCleanup()
					}
				}
			}()
			return nil
		},
		OnStop: func(_ context.Context) error {
			cancel()
			return nil
		},
	})
}

// runCleanup performs a single cleanup pass for expired page_data rows.
func runCleanup() {
	if storepkg.Database == nil || storepkg.Database.GetDB() == nil {
		return
	}
	client, ok := storepkg.Database.GetDB().(*storepkg.Client)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	store := storepkg.NewPageDataStore(client)
	count, err := store.DeleteExpiredPageData(ctx)
	if err != nil {
		flog.Err(fmt.Errorf("page_data cleanup: %w", err))
	} else if count > 0 {
		flog.Info("page_data cleanup: deleted %d expired rows", count)
	}
}
