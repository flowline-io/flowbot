package workflow

import (
	"context"
	"sync"
)

var (
	reloadMu      sync.Mutex
	reloadService *Service
)

// SetReloadService wires the package-level Service used by ReloadTriggers.
func SetReloadService(svc *Service) {
	reloadMu.Lock()
	defer reloadMu.Unlock()
	reloadService = svc
}

// ReloadTriggers reloads cron and webhook triggers on the wired Service.
func ReloadTriggers(ctx context.Context) error {
	reloadMu.Lock()
	svc := reloadService
	reloadMu.Unlock()
	if svc == nil {
		return nil
	}
	return svc.ReloadTriggers(ctx)
}

// ActiveService returns the package-level Service, or nil if not wired.
func ActiveService() *Service {
	reloadMu.Lock()
	defer reloadMu.Unlock()
	return reloadService
}
