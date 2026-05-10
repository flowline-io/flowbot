package command

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestAllLeafCommandsHaveRunE(t *testing.T) {
	commands := []func() *cobra.Command{
		LoginCommand,
		HubCommand,
		PipelineCommand,
		WorkflowCommand,
		BookmarkCommand,
		KanbanCommand,
		ReaderCommand,
		ConfigCommand,
		func() *cobra.Command { return VersionCommand("test") },
	}

	for _, fn := range commands {
		cmd := fn()
		checkAllLeavesHaveRunE(t, cmd, cmd.Name())
	}
}
