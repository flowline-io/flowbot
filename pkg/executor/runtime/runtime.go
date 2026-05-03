package runtime // import "https://github.com/runabol/tork"

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/types"
)

const (
	Docker     = "docker"
	Shell      = "shell"
	Machine    = "machine"
	Capability = "capability"
)

// Runtime is the actual runtime environment that executes a task.
type Runtime interface {
	Run(ctx context.Context, t *types.Task) error
	Stop(ctx context.Context, t *types.Task) error
	HealthCheck(ctx context.Context) error
	Close() error
}
