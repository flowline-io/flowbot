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
		ListDirTool{Workspace: ws, Env: executionEnv},
		GlobFilesTool{Workspace: ws, Env: executionEnv},
		GrepFilesTool{Workspace: ws, Env: executionEnv},
		ReadFileTool{Workspace: ws, Env: executionEnv},
		WriteFileTool{Workspace: ws, Env: executionEnv},
		ApplyPatchTool{Workspace: ws, Env: executionEnv},
		WebSearchTool{
			MaxOutput: ws.MaxOutput,
			APIKey:    ws.WebSearchAPIKey,
		},
		WebFetchTool{MaxOutput: ws.MaxOutput},
		RunCodeTool{Workspace: ws, Env: executionEnv},
	}
	for _, item := range tools {
		if err := registry.Register(item); err != nil {
			return err
		}
	}
	return nil
}

// HeadlessOptions configures RegisterHeadless tool selection.
type HeadlessOptions struct {
	// Force enables write and terminal tools (Cursor --force semantics).
	Force bool
}

// RegisterHeadless registers the flowbot-agent headless tool subset and returns
// their names for SetActive. Without Force: read-only tools. With Force: FS
// write tools plus run_terminal. Web and run_code tools are never registered.
func RegisterHeadless(registry *tool.Registry, ws Workspace, executionEnv env.ExecutionEnv, opts HeadlessOptions) ([]string, error) {
	if executionEnv == nil {
		executionEnv = env.Default()
	}
	tools := headlessTools(ws, executionEnv, opts.Force)
	names := make([]string, 0, len(tools))
	for _, item := range tools {
		if err := registry.Register(item); err != nil {
			return nil, err
		}
		names = append(names, item.Name())
	}
	return names, nil
}

// HeadlessToolNames returns active tool names for a headless run.
func HeadlessToolNames(force bool) []string {
	tools := headlessTools(Workspace{}, env.Default(), force)
	names := make([]string, 0, len(tools))
	for _, item := range tools {
		names = append(names, item.Name())
	}
	return names
}

func headlessTools(ws Workspace, executionEnv env.ExecutionEnv, force bool) []tool.Tool {
	tools := []tool.Tool{
		ListDirTool{Workspace: ws, Env: executionEnv},
		GlobFilesTool{Workspace: ws, Env: executionEnv},
		GrepFilesTool{Workspace: ws, Env: executionEnv},
		ReadFileTool{Workspace: ws, Env: executionEnv},
	}
	if force {
		tools = append(tools,
			WriteFileTool{Workspace: ws, Env: executionEnv},
			ApplyPatchTool{Workspace: ws, Env: executionEnv},
			RunTerminalTool{Workspace: ws, Env: executionEnv},
		)
	}
	return tools
}

// ActiveToolNames returns the default active coding tool names.
func ActiveToolNames() []string {
	return []string{
		"run_terminal",
		"list_dir",
		"glob_files",
		"grep_files",
		"read_file",
		"write_file",
		"apply_patch",
		"web_search",
		"web_fetch",
		"run_code",
	}
}
