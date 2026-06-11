package coding

import (
	"github.com/flowline-io/flowbot/pkg/agent/env"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
)

// RegisterAll registers chat assistant tools on the registry.
func RegisterAll(registry *tool.Registry, ws Workspace, executionEnv env.ExecutionEnv) error {
	if executionEnv == nil {
		executionEnv = env.Default()
	}
	tools := []tool.Tool{
		RunTerminalTool{Workspace: ws, Env: executionEnv},
		ReadFileTool{Workspace: ws, Env: executionEnv},
		WriteFileTool{Workspace: ws, Env: executionEnv},
		WebSearchTool{MaxOutput: ws.MaxOutput},
		RunCodeTool{Workspace: ws, Env: executionEnv},
	}
	for _, item := range tools {
		if err := registry.Register(item); err != nil {
			return err
		}
	}
	return nil
}

// ActiveToolNames returns the default active coding tool names.
func ActiveToolNames() []string {
	return []string{
		"run_terminal",
		"read_file",
		"write_file",
		"web_search",
		"run_code",
	}
}
