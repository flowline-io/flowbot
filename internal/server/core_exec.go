package server

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/pkg/agent/env"
	"github.com/flowline-io/flowbot/pkg/agent/sandbox"
	"github.com/flowline-io/flowbot/pkg/capability/core"
	"github.com/flowline-io/flowbot/pkg/config"
	pkgexec "github.com/flowline-io/flowbot/pkg/exec"
)

// coreExecProvider builds pkg/exec.Config from core/chat_agent workspace + sandbox.
type coreExecProvider struct{}

func (coreExecProvider) ExecConfig(_ context.Context) (pkgexec.Config, error) {
	ws := strings.TrimSpace(config.CoreWorkspace())
	if ws == "" {
		return pkgexec.Config{}, fmt.Errorf("core workspace is not configured (set core.workspace or chat_agent.workspace)")
	}
	timeout := config.App.ChatAgent.ShellTimeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	maxOut := config.App.ChatAgent.MaxToolOutput
	return pkgexec.Config{
		Workspace: ws,
		Env:       executionEnvForCore(ws),
		Timeout:   timeout,
		MaxOutput: maxOut,
	}, nil
}

func executionEnvForCore(workspace string) env.ExecutionEnv {
	cfg := config.App.ChatAgent.Sandbox
	if !cfg.Enabled {
		return env.Default()
	}
	return sandbox.New(sandbox.ConfigFromChatAgent(cfg, workspace), env.Default(), nil)
}

// wireCoreExecProvider installs the CapCore run_* execution backend.
func wireCoreExecProvider() {
	core.SetExecProvider(coreExecProvider{})
}
