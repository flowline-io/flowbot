package server

import (
	"context"
	"fmt"
	"time"

	storepkg "github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/flog"
)

// initPageDataCleanup starts a background goroutine that periodically deletes
// expired page_data rows.
func initPageDataCleanup() {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			if storepkg.Database == nil || storepkg.Database.GetDB() == nil {
				continue
			}
			client, ok := storepkg.Database.GetDB().(*storepkg.Client)
			if !ok {
				continue
			}
			store := storepkg.NewPageDataStore(client)
			count, err := store.DeleteExpiredPageData(context.Background())
			if err != nil {
				flog.Error(fmt.Errorf("page_data cleanup: %w", err))
			} else if count > 0 {
				flog.Info("page_data cleanup: deleted %d expired rows", count)
			}
		}
	}()
}
