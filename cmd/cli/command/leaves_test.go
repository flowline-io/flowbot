package command

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestAllLeafCommandsHaveRunE(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		fn   func() *cobra.Command
	}{
		{name: "login", fn: LoginCommand},
		{name: "hub", fn: HubCommand},
		{name: "pipeline", fn: PipelineCommand},
		{name: "workflow", fn: WorkflowCommand},
		{name: "bookmark", fn: BookmarkCommand},
		{name: "kanban", fn: KanbanCommand},
		{name: "reader", fn: ReaderCommand},
		{name: "memo", fn: MemoCommand},
		{name: "trilium", fn: TriliumCommand},
		{name: "fireflyiii", fn: FireflyiiiCommand},
		{name: "transmission", fn: TransmissionCommand},
		{name: "config", fn: ConfigCommand},
		{name: "version", fn: func() *cobra.Command { return VersionCommand("test") }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := tt.fn()
			checkAllLeavesHaveRunE(t, cmd, cmd.Name())
		})
	}
}

func checkAllLeavesHaveRunE(t *testing.T, cmd *cobra.Command, path string) {
	t.Helper()
	if !cmd.HasSubCommands() {
		if cmd.Name() == "" {
			return
		}
		require.NotNilf(t, cmd.RunE, "leaf command %q has no RunE", path)
		return
	}
	for _, sub := range cmd.Commands() {
		subPath := path + " " + sub.Name()
		checkAllLeavesHaveRunE(t, sub, subPath)
	}
}
