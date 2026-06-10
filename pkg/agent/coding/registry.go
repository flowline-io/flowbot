package coding

import (
	"github.com/flowline-io/flowbot/pkg/agent/tool"
)

// RegisterAll registers chat assistant tools on the registry.
func RegisterAll(registry *tool.Registry, ws Workspace) error {
	tools := []tool.Tool{
		RunTerminalTool{Workspace: ws},
		ReadFileTool{Workspace: ws},
		WriteFileTool{Workspace: ws},
		WebSearchTool{MaxOutput: ws.MaxOutput},
		RunCodeTool{Workspace: ws},
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
