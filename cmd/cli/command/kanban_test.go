package command

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestKanbanCommand(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "kanban command has correct use and subcommands"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := KanbanCommand()

			require.Equal(t, "kanban", cmd.Use)
			require.True(t, cmd.HasSubCommands())

			subNames := subcommandNames(cmd)
			require.Contains(t, subNames, "list")
			require.Contains(t, subNames, "search")
			require.Contains(t, subNames, "get")
			require.Contains(t, subNames, "create")
			require.Contains(t, subNames, "update")
			require.Contains(t, subNames, "delete")
			require.Contains(t, subNames, "move")
			require.Contains(t, subNames, "card")
			require.Contains(t, subNames, "column")
			require.Contains(t, subNames, "metadata")
			require.Contains(t, subNames, "tag")
			require.Contains(t, subNames, "subtask")
		})
	}
}

func TestKanbanListCommand(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "kanban list command has correct flags"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := KanbanCommand()
			listCmd := findSubcommand(cmd, "list")
			require.NotNil(t, listCmd)
			require.NotNil(t, listCmd.RunE)

			project := listCmd.Flags().Lookup("project")
			require.NotNil(t, project)
			val, _ := listCmd.Flags().GetInt("project")
			require.Equal(t, 1, val)

			status := listCmd.Flags().Lookup("status")
			require.NotNil(t, status)
			valStr, _ := listCmd.Flags().GetString("status")
			require.Equal(t, "active", valStr)
		})
	}
}

func TestKanbanCreateRequiredFlags(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "kanban create has required --title flag"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := KanbanCommand()
			createCmd := findSubcommand(cmd, "create")
			require.NotNil(t, createCmd)

			title := createCmd.Flags().Lookup("title")
			require.NotNil(t, title)
			ann := title.Annotations[cobra.BashCompOneRequiredFlag]
			require.NotNil(t, ann)
			require.Contains(t, ann, "true")
		})
	}
}

func TestKanbanMoveCommand(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "kanban move command has required --column flag"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := KanbanCommand()
			moveCmd := findSubcommand(cmd, "move")
			require.NotNil(t, moveCmd)
			require.NotNil(t, moveCmd.RunE)
			require.Contains(t, moveCmd.Use, "move")

			column := moveCmd.Flags().Lookup("column")
			require.NotNil(t, column)
			ann := column.Annotations[cobra.BashCompOneRequiredFlag]
			require.NotNil(t, ann)
			require.Contains(t, ann, "true")
		})
	}
}

func TestKanbanSubtaskCreateRequiredFlags(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "kanban subtask create has required flags"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := KanbanCommand()
			subtaskCmd := findSubcommand(cmd, "subtask")
			require.NotNil(t, subtaskCmd)
			createCmd := findSubcommand(subtaskCmd, "create")
			require.NotNil(t, createCmd)

			title := createCmd.Flags().Lookup("title")
			require.NotNil(t, title)
			ann := title.Annotations[cobra.BashCompOneRequiredFlag]
			require.NotNil(t, ann)

			status := createCmd.Flags().Lookup("status")
			require.NotNil(t, status)
			val, _ := createCmd.Flags().GetInt("status")
			require.Equal(t, 0, val)
		})
	}
}

func TestKanbanSubtaskTimerCommand(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "kanban subtask timer has correct subcommands"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := KanbanCommand()
			subtaskCmd := findSubcommand(cmd, "subtask")
			require.NotNil(t, subtaskCmd)
			timerCmd := findSubcommand(subtaskCmd, "timer")
			require.NotNil(t, timerCmd)

			subNames := subcommandNames(timerCmd)
			require.Contains(t, subNames, "check")
			require.Contains(t, subNames, "start")
			require.Contains(t, subNames, "stop")
			require.Contains(t, subNames, "spent")
		})
	}
}

func TestKanbanTagRequiredFlags(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "kanban tag create and update have required --name flag"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := KanbanCommand()
			tagCmd := findSubcommand(cmd, "tag")
			require.NotNil(t, tagCmd)

			createCmd := findSubcommand(tagCmd, "create")
			require.NotNil(t, createCmd)
			name := createCmd.Flags().Lookup("name")
			require.NotNil(t, name)
			ann := name.Annotations[cobra.BashCompOneRequiredFlag]
			require.NotNil(t, ann)

			updateCmd := findSubcommand(tagCmd, "update")
			require.NotNil(t, updateCmd)
			name = updateCmd.Flags().Lookup("name")
			require.NotNil(t, name)
			ann = name.Annotations[cobra.BashCompOneRequiredFlag]
			require.NotNil(t, ann)
		})
	}
}

func TestKanbanTaskTagSetRequiredFlags(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "kanban tag task set has required flags"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := KanbanCommand()
			tagCmd := findSubcommand(cmd, "tag")
			taskCmd := findSubcommand(tagCmd, "task")
			require.NotNil(t, taskCmd)
			setCmd := findSubcommand(taskCmd, "set")
			require.NotNil(t, setCmd)

			project := setCmd.Flags().Lookup("project")
			require.NotNil(t, project)
			ann := project.Annotations[cobra.BashCompOneRequiredFlag]
			require.NotNil(t, ann)

			tags := setCmd.Flags().Lookup("tags")
			require.NotNil(t, tags)
			ann = tags.Annotations[cobra.BashCompOneRequiredFlag]
			require.NotNil(t, ann)
		})
	}
}

func TestKanbanCardCommand(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "kanban card has correct subcommands"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := KanbanCommand()
			cardCmd := findSubcommand(cmd, "card")
			require.NotNil(t, cardCmd)

			subNames := subcommandNames(cardCmd)
			require.Contains(t, subNames, "add")
			require.Contains(t, subNames, "move")
			require.Contains(t, subNames, "delete")
		})
	}
}

func TestKanbanCardAddRequiredFlags(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "kanban card add has required --title flag"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := KanbanCommand()
			cardCmd := findSubcommand(cmd, "card")
			addCmd := findSubcommand(cardCmd, "add")
			require.NotNil(t, addCmd)

			title := addCmd.Flags().Lookup("title")
			require.NotNil(t, title)
			ann := title.Annotations[cobra.BashCompOneRequiredFlag]
			require.NotNil(t, ann)
		})
	}
}

func TestKanbanCardMoveRequiredFlags(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "kanban card move has required --column flag"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := KanbanCommand()
			cardCmd := findSubcommand(cmd, "card")
			moveCmd := findSubcommand(cardCmd, "move")
			require.NotNil(t, moveCmd)

			column := moveCmd.Flags().Lookup("column")
			require.NotNil(t, column)
			ann := column.Annotations[cobra.BashCompOneRequiredFlag]
			require.NotNil(t, ann)
		})
	}
}

func TestKanbanColumnListCommand(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "kanban column list has correct flags"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := KanbanCommand()
			columnCmd := findSubcommand(cmd, "column")
			require.NotNil(t, columnCmd)
			listCmd := findSubcommand(columnCmd, "list")
			require.NotNil(t, listCmd)
			require.NotNil(t, listCmd.RunE)

			project := listCmd.Flags().Lookup("project")
			require.NotNil(t, project)
		})
	}
}

func TestKanbanMetadataCommand(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "kanban metadata has correct subcommands"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := KanbanCommand()
			metaCmd := findSubcommand(cmd, "metadata")
			require.NotNil(t, metaCmd)

			subNames := subcommandNames(metaCmd)
			require.Contains(t, subNames, "get")
			require.Contains(t, subNames, "set")
			require.Contains(t, subNames, "delete")
		})
	}
}

func TestKanbanConfirmCommands(t *testing.T) {
	tests := []struct {
		parent string
		leaf   string
	}{
		{parent: "kanban", leaf: "delete"},
		{parent: "subtask", leaf: "delete"},
		{parent: "metadata", leaf: "delete"},
		{parent: "tag", leaf: "delete"},
		{parent: "card", leaf: "delete"},
	}

	for _, tt := range tests {
		t.Run(strings.Join([]string{tt.parent, tt.leaf}, "/"), func(t *testing.T) {
			cmd := KanbanCommand()

			parentCmd := cmd
			if tt.parent != "kanban" {
				parentCmd = findSubcommand(cmd, tt.parent)
				require.NotNil(t, parentCmd, "parent %s not found", tt.parent)
			}
			leafCmd := findSubcommand(parentCmd, tt.leaf)
			require.NotNil(t, leafCmd, "leaf %s not found under %s", tt.leaf, tt.parent)
			require.NotNil(t, leafCmd.RunE)

			yes := leafCmd.Flags().Lookup("yes")
			require.NotNil(t, yes, "--yes flag missing on %s/%s", tt.parent, tt.leaf)
			require.Equal(t, "y", yes.Shorthand)
		})
	}
}
