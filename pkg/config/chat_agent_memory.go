package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultMemoryFile     = "MEMORIES.md"
	defaultMemoryMaxBytes = 65536
	defaultMemoryDirName  = "agent-memories"
)

// ChatAgentDefaultMemoryFile returns the default memory filename.
func ChatAgentDefaultMemoryFile() string {
	return defaultMemoryFile
}

// ChatAgentMemoryMaxFileBytes returns the max memory file size in bytes.
func ChatAgentMemoryMaxFileBytes() int {
	return defaultMemoryMaxBytes
}

// MemoryDirectory resolves the absolute memory storage root and ensures it exists.
// Files are stored at <workspace-parent>/agent-memories/{scope}/*.md.
func MemoryDirectory() (string, error) {
	dir, err := resolveMemoryDirectory()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("memory directory: %w", err)
	}
	return dir, nil
}

func resolveMemoryDirectory() (string, error) {
	workspace := strings.TrimSpace(App.ChatAgent.Workspace)
	if workspace == "" {
		return "", fmt.Errorf("chat_agent.workspace is required for agent memory storage")
	}
	workspaceAbs, err := filepath.Abs(workspace)
	if err != nil {
		return "", fmt.Errorf("chat_agent.workspace: %w", err)
	}
	return filepath.Join(filepath.Dir(workspaceAbs), defaultMemoryDirName), nil
}
