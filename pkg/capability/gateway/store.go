package gateway

import (
	"context"
	"sync"
	"time"

	"github.com/flowline-io/flowbot/pkg/types"
)

// JobStore is the persistence port for CapGateway (injected from internal/server).
type JobStore interface {
	Create(ctx context.Context, in types.GatewayCreateJob) (*types.GatewayJob, error)
	Get(ctx context.Context, jobID string) (*types.GatewayJob, error)
	Cancel(ctx context.Context, jobID string) (*types.GatewayJob, error)
	HasFreshWorker(ctx context.Context, staleAfter time.Duration) (bool, error)
	ReclaimExpired(ctx context.Context) error
}

var (
	storeMu sync.RWMutex
	jobs    JobStore
)

// SetJobStore injects the gateway job store used by CapGateway handlers.
func SetJobStore(s JobStore) {
	storeMu.Lock()
	defer storeMu.Unlock()
	jobs = s
}

func jobStore() JobStore {
	storeMu.RLock()
	defer storeMu.RUnlock()
	return jobs
}
