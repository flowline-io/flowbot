package core

import (
	"context"
	"strings"
	"sync"

	"github.com/flowline-io/flowbot/pkg/agent/dcg"
	"github.com/flowline-io/flowbot/pkg/agent/env"
	"github.com/flowline-io/flowbot/pkg/capability"
	pkgexec "github.com/flowline-io/flowbot/pkg/exec"
	"github.com/flowline-io/flowbot/pkg/types"
)

// ExecProvider supplies workspace-bound execution config for run_code / run_terminal.
type ExecProvider interface {
	// ExecConfig returns workspace root and ExecutionEnv (typically Docker sandbox).
	ExecConfig(ctx context.Context) (pkgexec.Config, error)
}

var (
	execMu       sync.RWMutex
	execProvider ExecProvider
)

// SetExecProvider wires sandboxed execution used by run_code and run_terminal.
func SetExecProvider(p ExecProvider) {
	execMu.Lock()
	defer execMu.Unlock()
	execProvider = p
}

func getExecProvider() ExecProvider {
	execMu.RLock()
	defer execMu.RUnlock()
	return execProvider
}

func runTerminalInvoker(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
	command, err := capability.RequiredString(params, "command")
	if err != nil {
		return nil, err
	}
	workdir, _ := capability.StringParam(params, "workdir")
	if err := dcgGuardTerminal(ctx, command); err != nil {
		return nil, err
	}
	cfg, err := resolveExecConfig(ctx)
	if err != nil {
		return nil, err
	}
	res, err := pkgexec.RunTerminal(ctx, cfg, command, workdir)
	if err != nil {
		return nil, err
	}
	return execInvokeResult(res), nil
}

func runCodeInvoker(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
	language, err := capability.RequiredString(params, "language")
	if err != nil {
		return nil, err
	}
	code, err := capability.RequiredString(params, "code")
	if err != nil {
		return nil, err
	}
	filename, _ := capability.StringParam(params, "filename")
	workdir, _ := capability.StringParam(params, "workdir")
	if err := dcgGuardCode(ctx, language, code); err != nil {
		return nil, err
	}
	cfg, err := resolveExecConfig(ctx)
	if err != nil {
		return nil, err
	}
	res, err := pkgexec.RunCode(ctx, cfg, language, code, filename, workdir)
	if err != nil {
		return nil, err
	}
	return execInvokeResult(res), nil
}

func execInvokeResult(res pkgexec.Result) *capability.InvokeResult {
	return &capability.InvokeResult{
		Data: map[string]any{
			"stdout":    res.Stdout,
			"stderr":    res.Stderr,
			"exit_code": res.ExitCode,
			"output":    res.Output,
		},
		Text: res.Output,
	}
}

func resolveExecConfig(ctx context.Context) (pkgexec.Config, error) {
	p := getExecProvider()
	if p == nil {
		return pkgexec.Config{}, types.Errorf(types.ErrUnavailable, "exec provider is not configured")
	}
	cfg, err := p.ExecConfig(ctx)
	if err != nil {
		return pkgexec.Config{}, err
	}
	if strings.TrimSpace(cfg.Workspace) == "" {
		return pkgexec.Config{}, types.Errorf(types.ErrUnavailable, "workspace is not configured")
	}
	if cfg.Env == nil {
		cfg.Env = env.Default()
	}
	return cfg, nil
}

func dcgGuardTerminal(ctx context.Context, command string) error {
	decision, err := dcg.DefaultChecker().Check(ctx, command)
	if err != nil {
		return types.Errorf(types.ErrUnavailable, "dcg check failed: %v", err)
	}
	if !decision.Allow {
		reason := decision.Reason
		if reason == "" {
			reason = dcg.ReasonBlocked
		}
		return types.Errorf(types.ErrForbidden, "%s", reason)
	}
	return nil
}

func dcgGuardCode(ctx context.Context, language, code string) error {
	synth, err := dcg.SynthCommand(language, code)
	if err != nil {
		return types.Errorf(types.ErrInvalidArgument, "%v", err)
	}
	return dcgGuardTerminal(ctx, synth)
}
