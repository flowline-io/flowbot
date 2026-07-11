package memory

import (
	"github.com/flowline-io/flowbot/pkg/config"
)

// OpenFromConfig opens the memory store under <workspace-parent>/agent-memories.
func OpenFromConfig() (*FileStore, error) {
	dir, err := config.MemoryDirectory()
	if err != nil {
		return nil, err
	}
	return NewFileStore(dir, config.ChatAgentDefaultMemoryFile(), config.ChatAgentMemoryMaxFileBytes())
}
