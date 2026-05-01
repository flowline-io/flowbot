package homelab

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/types"
)

type Runtime interface {
	Status(ctx context.Context, app App) (AppStatus, error)
	Logs(ctx context.Context, app App, tail int) ([]string, error)
	Start(ctx context.Context, app App) error
	Stop(ctx context.Context, app App) error
	Restart(ctx context.Context, app App) error
	Pull(ctx context.Context, app App) error
	Update(ctx context.Context, app App) error
}

type NoopRuntime struct{}

func NewRuntime(config RuntimeConfig, appsDir string) Runtime {
	switch config.Mode {
	case RuntimeModeDockerSocket:
		return NewDockerComposeRuntime(config, appsDir)
	case RuntimeModeSSH:
		return NewSSHRuntime(config)
	default:
		return NoopRuntime{}
	}
}

func (NoopRuntime) Status(ctx context.Context, app App) (AppStatus, error) {
	if err := ctx.Err(); err != nil {
		return AppStatusUnknown, types.WrapError(types.ErrTimeout, "homelab status canceled", err)
	}
	return app.Status, nil
}

func (NoopRuntime) Logs(ctx context.Context, app App, tail int) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "homelab logs canceled", err)
	}
	return nil, types.Errorf(types.ErrNotImplemented, "homelab runtime logs are not implemented")
}

func (NoopRuntime) Start(ctx context.Context, app App) error {
	return notImplemented(ctx, "start")
}

func (NoopRuntime) Stop(ctx context.Context, app App) error {
	return notImplemented(ctx, "stop")
}

func (NoopRuntime) Restart(ctx context.Context, app App) error {
	return notImplemented(ctx, "restart")
}

func (NoopRuntime) Pull(ctx context.Context, app App) error {
	return notImplemented(ctx, "pull")
}

func (NoopRuntime) Update(ctx context.Context, app App) error {
	return notImplemented(ctx, "update")
}

func notImplemented(ctx context.Context, operation string) error {
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "homelab operation canceled", err)
	}
	return types.Errorf(types.ErrNotImplemented, "homelab runtime %s is not implemented", operation)
}
