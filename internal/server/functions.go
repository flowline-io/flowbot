package server

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"go.uber.org/fx"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/agent/env"
	"github.com/flowline-io/flowbot/pkg/agent/result"
	"github.com/flowline-io/flowbot/pkg/agent/sandbox"
	capfunctions "github.com/flowline-io/flowbot/pkg/capability/functions"
	"github.com/flowline-io/flowbot/pkg/config"
	pkgexec "github.com/flowline-io/flowbot/pkg/exec"
	"github.com/flowline-io/flowbot/pkg/flog"
	pkgfunctions "github.com/flowline-io/flowbot/pkg/functions"
	"github.com/flowline-io/flowbot/pkg/types"
)

// functionExecProvider builds a sandbox exec config with Network=none and no Flowbot credentials.
// Workspace is set per Invoke by Service; Exec uses opts.Dir as the bind-mounted workspace.
type functionExecProvider struct{}

var _ pkgfunctions.WorkspacePreparer = functionExecProvider{}

func (functionExecProvider) PrepareWorkspace(workspace string) error {
	return sandbox.EnsureAgentReadable(workspace)
}

func (functionExecProvider) ExecConfig(_ context.Context) (pkgexec.Config, error) {
	sb := config.App.ChatAgent.Sandbox
	if !sb.Enabled {
		return pkgexec.Config{}, types.Errorf(types.ErrUnavailable, "function sandbox is disabled (chat_agent.sandbox.enabled=false)")
	}
	image := strings.TrimSpace(sb.Image)
	if image == "" {
		image = "ghcr.io/flowline-io/flowbot-agent-sandbox:latest"
	}
	return pkgexec.Config{
		Env: &functionSandboxEnv{
			host:    env.Default(),
			image:   image,
			memory:  strings.TrimSpace(sb.Memory),
			network: "none",
			runtime: strings.TrimSpace(sb.Runtime),
			profile: strings.TrimSpace(sb.SecurityProfile),
		},
		Timeout:   pkgfunctions.DefaultTimeout,
		MaxOutput: pkgfunctions.MaxJSONBytes * 2,
	}, nil
}

// functionSandboxEnv uses host FS for file ops and a sandbox (docker or kern) for Exec.
// Each Exec binds opts.Dir as the workspace (Service creates an ephemeral dir per Invoke).
type functionSandboxEnv struct {
	host    env.ExecutionEnv
	image   string
	memory  string
	network string
	runtime string
	profile string
}

func (e *functionSandboxEnv) ReadFile(ctx context.Context, path string) result.Result[[]byte, result.FileError] {
	return e.host.ReadFile(ctx, path)
}

func (e *functionSandboxEnv) WriteFile(ctx context.Context, path string, data []byte, perm os.FileMode) result.Result[struct{}, result.FileError] {
	res := e.host.WriteFile(ctx, path, data, perm)
	if !res.IsOk() {
		return res
	}
	if err := sandbox.EnsureAgentReadable(path); err != nil {
		return result.Err[struct{}, result.FileError](result.NewFileError("chmod", path, err))
	}
	return res
}

func (e *functionSandboxEnv) MkdirAll(ctx context.Context, path string, perm os.FileMode) result.Result[struct{}, result.FileError] {
	res := e.host.MkdirAll(ctx, path, perm)
	if !res.IsOk() {
		return res
	}
	if err := sandbox.EnsureAgentReadable(path); err != nil {
		return result.Err[struct{}, result.FileError](result.NewFileError("chmod", path, err))
	}
	return res
}

func (e *functionSandboxEnv) Remove(ctx context.Context, path string) result.Result[struct{}, result.FileError] {
	return e.host.Remove(ctx, path)
}

func (e *functionSandboxEnv) ReadDir(ctx context.Context, path string) result.Result[[]env.DirEntry, result.FileError] {
	return e.host.ReadDir(ctx, path)
}

func (e *functionSandboxEnv) Exec(ctx context.Context, opts env.ExecOptions) result.Result[env.Capture, result.ExecutionError] {
	workspace := strings.TrimSpace(opts.Dir)
	if workspace == "" {
		return result.Err[env.Capture, result.ExecutionError](
			result.NewExecutionError("spawn_error", "function sandbox workspace is required", fmt.Errorf("empty workspace")),
		)
	}
	sb := sandbox.New(sandbox.Config{
		Runtime:         e.runtime,
		SecurityProfile: e.profile,
		Image:           e.image,
		Network:         e.network,
		Memory:          e.memory,
		Workspace:       workspace,
		ServerURL:       "",
		AccessToken:     "",
	}, e.host, nil)
	return sb.Exec(ctx, opts)
}

func initFunctions(lc fx.Lifecycle) error {
	if store.Database == nil || store.Database.GetClient() == nil {
		flog.Warn("functions: database not ready, skipping service wiring")
		return nil
	}
	catalog := store.NewFunctionCatalogAdapter(store.FunctionStoreFromDB())
	svc := pkgfunctions.NewService(catalog, functionExecProvider{})
	pkgfunctions.SetActiveService(svc)

	if err := capfunctions.Register(); err != nil {
		pkgfunctions.SetActiveService(nil)
		return fmt.Errorf("register functions capability: %w", err)
	}

	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			pkgfunctions.SetActiveService(nil)
			return nil
		},
	})

	flog.Info("functions service initialized (timeout=%s)", time.Duration(pkgfunctions.DefaultTimeout))
	return nil
}
