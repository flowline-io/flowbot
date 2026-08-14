package functions

import (
	"context"

	pkgexec "github.com/flowline-io/flowbot/pkg/exec"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

// Catalog loads and mutates named function definitions and runs.
type Catalog interface {
	Create(ctx context.Context, name, metadata, entrypoint, source, createdBy string) error
	GetByName(ctx context.Context, name string) (*model.FunctionDefinition, error)
	UpdateDraft(ctx context.Context, name, metadata, entrypoint, source string, version int) (*model.FunctionDefinition, error)
	Publish(ctx context.Context, name string, version int) (*model.FunctionDefinition, error)
	Delete(ctx context.Context, name string) (int64, error)
	ListPublished(ctx context.Context) ([]*model.FunctionDefinition, error)
	ListAll(ctx context.Context) ([]*model.FunctionDefinition, error)
	GetVersion(ctx context.Context, name string, version int) (*model.FunctionDefinitionVersion, error)
	GetLatestPublished(ctx context.Context, name string) (*model.FunctionDefinitionVersion, error)
	CreateRun(ctx context.Context, name string, version int, idempotencyKey *string) (*model.FunctionRun, error)
	GetRunByIdempotency(ctx context.Context, name, key string) (*model.FunctionRun, error)
	CompleteRun(ctx context.Context, id int64, status string, durationMs int64, exitCode *int, errMsg string, resultJSON *string) (*model.FunctionRun, error)
	ListRuns(ctx context.Context, name string) ([]*model.FunctionRun, error)
}

// ExecProvider supplies an execution Config for ephemeral function runs.
type ExecProvider interface {
	ExecConfig(ctx context.Context) (pkgexec.Config, error)
}
