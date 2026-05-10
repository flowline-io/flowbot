package command

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/cmd/cli/utils"
	"github.com/flowline-io/flowbot/pkg/client"
	"github.com/flowline-io/flowbot/pkg/providers/kanboard"
)

func KanbanCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kanban",
		Short: "Work with kanban boards",
		Long:  "Manage kanban boards via Flowbot server",
	}
	cmd.AddCommand(
		kanbanListCommand(),
		kanbanSearchCommand(),
		kanbanGetCommand(),
		kanbanCreateCommand(),
		kanbanUpdateCommand(),
		kanbanDeleteCommand(),
		kanbanMoveCommand(),
		kanbanCardCommand(),
		kanbanColumnCommand(),
		kanbanMetadataCommand(),
		kanbanTagCommand(),
		kanbanSubtaskCommand(),
	)
	return cmd
}

func kanbanListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all kanban tasks",
		Long:  "Display kanban tasks from Flowbot server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			projectId, _ := cmd.Flags().GetInt("project")
			status, _ := cmd.Flags().GetString("status")

			var tasks []kanboard.Task
			if status == "all" {
				tasks, err = c.Kanban.ListAll(cmd.Context(), projectId)
			} else {
				statusId := kanboard.Active
				if status == "inactive" {
					statusId = kanboard.Inactive
				}
				tasks, err = c.Kanban.List(cmd.Context(), projectId, statusId)
			}
			if err != nil {
				return fmt.Errorf("list kanban tasks: %w", err)
			}

			if len(tasks) == 0 {
				_, _ = fmt.Println("No kanban tasks found")
				return nil
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				data, err := sonic.MarshalIndent(tasks, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal kanban tasks: %w", err)
				}
				_, _ = fmt.Println(string(data))
			} else {
				_, _ = fmt.Printf("%-8s %-30s %-15s %-10s\n", "ID", "TITLE", "COLUMN", "STATUS")
				_, _ = fmt.Println(strings.Repeat("-", 65))
				for i := range tasks {
					id := strconv.Itoa(tasks[i].ID)
					title := tasks[i].Title
					if len(title) > 28 {
						title = title[:25] + "..."
					}
					column := tasks[i].ColumnTitle
					if len(column) > 13 {
						column = column[:10] + "..."
					}
					statusStr := "active"
					if tasks[i].IsActive == 0 {
						statusStr = "closed"
					}
					_, _ = fmt.Printf("%-8s %-30s %-15s %-10s\n", id, title, column, statusStr)
				}
			}

			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	cmd.Flags().IntP("project", "p", 1, "Project ID")
	cmd.Flags().StringP("status", "s", "active", "Status filter (active, inactive, all)")
	return cmd
}

func kanbanGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get a kanban task by ID",
		Long:  "Display details of a specific kanban task",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("task ID is required")
			}
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			task, err := c.Kanban.Get(cmd.Context(), id)
			if err != nil {
				return fmt.Errorf("get kanban task: %w", err)
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				data, err := sonic.MarshalIndent(task, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal kanban task: %w", err)
				}
				_, _ = fmt.Println(string(data))
			} else {
				_, _ = fmt.Printf("ID:          %d\n", task.ID)
				_, _ = fmt.Printf("Title:       %s\n", task.Title)
				_, _ = fmt.Printf("Description: %s\n", task.Description)
				_, _ = fmt.Printf("Project:     %s\n", task.ProjectName)
				_, _ = fmt.Printf("Column:      %s\n", task.ColumnTitle)
				_, _ = fmt.Printf("Priority:    %d\n", task.Priority)
				_, _ = fmt.Printf("Status:      %s\n", map[int]string{0: "inactive", 1: "active"}[task.IsActive])
				_, _ = fmt.Printf("Created:     %s\n", formatTimestamp(task.DateCreation))
				_, _ = fmt.Printf("Updated:     %s\n", formatTimestamp(task.DateModification))
			}

			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func kanbanCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new kanban task",
		Long:  "Add a new task to the kanban board",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			title, _ := cmd.Flags().GetString("title")
			description, _ := cmd.Flags().GetString("description")
			projectId, _ := cmd.Flags().GetInt("project")
			columnId, _ := cmd.Flags().GetInt("column")

			req := client.KanbanCreateRequest{
				Title:       title,
				Description: description,
				ProjectID:   projectId,
				ColumnID:    columnId,
			}

			result, err := c.Kanban.Create(cmd.Context(), req)
			if err != nil {
				return fmt.Errorf("create kanban task: %w", err)
			}

			_, _ = fmt.Printf("Task created: ID=%d\n", result.ID)
			return nil
		},
	}
	cmd.Flags().StringP("title", "t", "", "Task title")
	_ = cmd.MarkFlagRequired("title")
	cmd.Flags().StringP("description", "d", "", "Task description")
	cmd.Flags().IntP("project", "p", 1, "Project ID")
	cmd.Flags().IntP("column", "c", 0, "Column ID")
	return cmd
}

func kanbanUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a kanban task",
		Long:  "Modify an existing kanban task",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("task ID is required")
			}
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			req := client.KanbanUpdateRequest{}
			if title, _ := cmd.Flags().GetString("title"); title != "" {
				req.Title = title
			}
			if desc, _ := cmd.Flags().GetString("description"); desc != "" {
				req.Description = desc
			}

			_, err = c.Kanban.Update(cmd.Context(), id, req)
			if err != nil {
				return fmt.Errorf("update kanban task: %w", err)
			}

			_, _ = fmt.Printf("Task updated: %d\n", id)
			return nil
		},
	}
	cmd.Flags().StringP("title", "t", "", "New title")
	cmd.Flags().StringP("description", "d", "", "New description")
	return cmd
}

func kanbanDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Close a kanban task",
		Long:  "Close a task by ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("task ID is required")
			}
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			yes, _ := cmd.Flags().GetBool("yes")
			if !yes {
				_, _ = fmt.Printf("Close task %d? [y/N]: ", id)
				var response string
				if _, err := fmt.Scanln(&response); err != nil {
					return fmt.Errorf("read confirmation: %w", err)
				}
				if response != "y" && response != "Y" {
					_, _ = fmt.Println("Cancelled")
					return nil
				}
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			_, err = c.Kanban.Close(cmd.Context(), id)
			if err != nil {
				return fmt.Errorf("close kanban task: %w", err)
			}

			_, _ = fmt.Printf("Task closed: %d\n", id)
			return nil
		},
	}
	cmd.Flags().BoolP("yes", "y", false, "Skip confirmation")
	return cmd
}

func kanbanMoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "move <id>",
		Short: "Move a kanban task to another column",
		Long:  "Move a task to a different column",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("task ID is required")
			}
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			column, _ := cmd.Flags().GetInt("column")
			position, _ := cmd.Flags().GetInt("position")
			project, _ := cmd.Flags().GetInt("project")

			req := client.KanbanMoveRequest{
				ColumnID:  column,
				Position:  position,
				ProjectID: project,
			}

			_, err = c.Kanban.Move(cmd.Context(), id, req)
			if err != nil {
				return fmt.Errorf("move kanban task: %w", err)
			}

			_, _ = fmt.Printf("Task moved: %d -> column %d\n", id, req.ColumnID)
			return nil
		},
	}
	cmd.Flags().IntP("column", "c", 0, "Destination column ID")
	_ = cmd.MarkFlagRequired("column")
	cmd.Flags().IntP("position", "p", 0, "Position in column (0 = first)")
	cmd.Flags().IntP("project", "r", 1, "Project ID")
	return cmd
}

func kanbanSearchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search kanban tasks",
		Long:  "Search tasks using kanboard search syntax",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("search query is required")
			}
			query := args[0]

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			projectId, _ := cmd.Flags().GetInt("project")
			tasks, err := c.Kanban.Search(cmd.Context(), projectId, query)
			if err != nil {
				return fmt.Errorf("search kanban tasks: %w", err)
			}

			if len(tasks) == 0 {
				_, _ = fmt.Println("No tasks found")
				return nil
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				data, err := sonic.MarshalIndent(tasks, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal tasks: %w", err)
				}
				_, _ = fmt.Println(string(data))
			} else {
				_, _ = fmt.Printf("%-8s %-30s %-15s %-10s\n", "ID", "TITLE", "COLUMN", "STATUS")
				_, _ = fmt.Println(strings.Repeat("-", 65))
				for i := range tasks {
					id := strconv.Itoa(tasks[i].ID)
					title := tasks[i].Title
					if len(title) > 28 {
						title = title[:25] + "..."
					}
					column := tasks[i].ColumnTitle
					if len(column) > 13 {
						column = column[:10] + "..."
					}
					statusStr := "active"
					if tasks[i].IsActive == 0 {
						statusStr = "closed"
					}
					_, _ = fmt.Printf("%-8s %-30s %-15s %-10s\n", id, title, column, statusStr)
				}
			}

			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	cmd.Flags().IntP("project", "p", 1, "Project ID")
	return cmd
}

func kanbanMetadataCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metadata",
		Short: "Manage task metadata",
		Long:  "Get, set, or delete task metadata",
	}
	cmd.AddCommand(
		kanbanMetadataGetCommand(),
		kanbanMetadataSetCommand(),
		kanbanMetadataDeleteCommand(),
	)
	return cmd
}

func kanbanMetadataGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <task_id> [name]",
		Short: "Get task metadata",
		Long:  "Get all metadata or a specific metadata value by name",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("task ID is required")
			}
			taskId, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			if len(args) > 1 {
				name := args[1]
				value, err := c.Kanban.GetMetadataByName(cmd.Context(), taskId, name)
				if err != nil {
					return fmt.Errorf("get metadata: %w", err)
				}
				_, _ = fmt.Println(value)
			} else {
				metadata, err := c.Kanban.GetMetadata(cmd.Context(), taskId)
				if err != nil {
					return fmt.Errorf("get metadata: %w", err)
				}
				data, err := sonic.MarshalIndent(metadata, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal metadata: %w", err)
				}
				_, _ = fmt.Println(string(data))
			}

			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "json", "Output format (json, value)")
	return cmd
}

func kanbanMetadataSetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <task_id> <name=value>...",
		Short: "Set task metadata",
		Long:  "Set one or more metadata values for a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("task ID and at least one name=value pair are required")
			}
			taskId, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			values := make(kanboard.TaskMetadata)
			for i := 1; i < len(args); i++ {
				parts := strings.SplitN(args[i], "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid format for %s, expected name=value", args[i])
				}
				values[parts[0]] = parts[1]
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			result, err := c.Kanban.SaveMetadata(cmd.Context(), taskId, values)
			if err != nil {
				return fmt.Errorf("save metadata: %w", err)
			}

			if result.Success {
				_, _ = fmt.Println("Metadata saved successfully")
			} else {
				_, _ = fmt.Println("Failed to save metadata")
			}

			return nil
		},
	}
	return cmd
}

func kanbanMetadataDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <task_id> <name>",
		Short: "Delete task metadata",
		Long:  "Delete a metadata entry from a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("task ID and metadata name are required")
			}
			taskId, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}
			name := args[1]

			yes, _ := cmd.Flags().GetBool("yes")
			if !yes {
				_, _ = fmt.Printf("Delete metadata '%s' from task %d? [y/N]: ", name, taskId)
				var response string
				if _, err := fmt.Scanln(&response); err != nil {
					return fmt.Errorf("read confirmation: %w", err)
				}
				if response != "y" && response != "Y" {
					_, _ = fmt.Println("Cancelled")
					return nil
				}
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			result, err := c.Kanban.RemoveMetadata(cmd.Context(), taskId, name)
			if err != nil {
				return fmt.Errorf("remove metadata: %w", err)
			}

			if result.Success {
				_, _ = fmt.Printf("Metadata '%s' deleted from task %d\n", name, taskId)
			} else {
				_, _ = fmt.Println("Failed to delete metadata")
			}

			return nil
		},
	}
	cmd.Flags().BoolP("yes", "y", false, "Skip confirmation")
	return cmd
}

func formatTimestamp(ts int) string {
	if ts == 0 {
		return "N/A"
	}
	t := time.Unix(int64(ts), 0)
	return t.Format(time.RFC3339)
}

func kanbanSubtaskCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subtask",
		Short: "Manage kanban subtasks",
		Long:  "Create, update, list and delete subtasks for kanban tasks",
	}
	cmd.AddCommand(
		kanbanSubtaskListCommand(),
		kanbanSubtaskGetCommand(),
		kanbanSubtaskCreateCommand(),
		kanbanSubtaskUpdateCommand(),
		kanbanSubtaskDeleteCommand(),
		kanbanSubtaskTimerCommand(),
	)
	return cmd
}

func kanbanSubtaskListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <task_id>",
		Short: "List subtasks for a task",
		Long:  "Display all subtasks for a given task",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("task ID is required")
			}
			taskId, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			subtasks, err := c.Kanban.ListSubtasks(cmd.Context(), taskId)
			if err != nil {
				return fmt.Errorf("list subtasks: %w", err)
			}

			if len(subtasks) == 0 {
				_, _ = fmt.Println("No subtasks found for this task")
				return nil
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				data, err := sonic.MarshalIndent(subtasks, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal subtasks: %w", err)
				}
				_, _ = fmt.Println(string(data))
			} else {
				_, _ = fmt.Printf("%-8s %-30s %-10s %-12s %-12s\n", "ID", "TITLE", "STATUS", "ESTIMATED", "SPENT")
				_, _ = fmt.Println(strings.Repeat("-", 80))
				for _, s := range subtasks {
					title := s.Title
					if len(title) > 28 {
						title = title[:25] + "..."
					}
					status := s.StatusName
					if status == "" {
						status = "Todo"
					}
					_, _ = fmt.Printf("%-8s %-30s %-10s %-12s %-12s\n", s.ID, title, status, s.TimeEstimated, s.TimeSpent)
				}
			}

			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func kanbanSubtaskGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <task_id> <subtask_id>",
		Short: "Get a subtask by ID",
		Long:  "Display details of a specific subtask",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("task ID and subtask ID are required")
			}
			taskId, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}
			subtaskId, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid subtask ID: %w", err)
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			subtask, err := c.Kanban.GetSubtask(cmd.Context(), taskId, subtaskId)
			if err != nil {
				return fmt.Errorf("get subtask: %w", err)
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				data, err := sonic.MarshalIndent(subtask, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal subtask: %w", err)
				}
				_, _ = fmt.Println(string(data))
			} else {
				_, _ = fmt.Printf("ID:          %s\n", subtask.ID)
				_, _ = fmt.Printf("Title:       %s\n", subtask.Title)
				_, _ = fmt.Printf("Task ID:     %s\n", subtask.TaskID)
				_, _ = fmt.Printf("Status:      %s\n", subtask.StatusName)
				_, _ = fmt.Printf("Time Estimated: %v\n", subtask.TimeEstimated)
				_, _ = fmt.Printf("Time Spent:  %v\n", subtask.TimeSpent)
				if subtask.Username != "" {
					_, _ = fmt.Printf("Assignee:    %s\n", subtask.Username)
				}
			}

			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func kanbanSubtaskCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <task_id>",
		Short: "Create a new subtask",
		Long:  "Add a subtask to a kanban task",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("task ID is required")
			}
			taskId, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			title, _ := cmd.Flags().GetString("title")
			user, _ := cmd.Flags().GetInt("user")
			timeEstimated, _ := cmd.Flags().GetInt("time-estimated")
			timeSpent, _ := cmd.Flags().GetInt("time-spent")
			status, _ := cmd.Flags().GetInt("status")

			req := client.KanbanCreateSubtaskRequest{
				Title:         title,
				UserID:        user,
				TimeEstimated: timeEstimated,
				TimeSpent:     timeSpent,
				Status:        status,
			}

			result, err := c.Kanban.CreateSubtask(cmd.Context(), taskId, req)
			if err != nil {
				return fmt.Errorf("create subtask: %w", err)
			}

			_, _ = fmt.Printf("Subtask created: ID=%d\n", result.ID)
			return nil
		},
	}
	cmd.Flags().StringP("title", "t", "", "Subtask title")
	_ = cmd.MarkFlagRequired("title")
	cmd.Flags().IntP("user", "u", 0, "User ID to assign")
	cmd.Flags().IntP("time-estimated", "e", 0, "Estimated time (minutes)")
	cmd.Flags().IntP("time-spent", "s", 0, "Time spent (minutes)")
	cmd.Flags().IntP("status", "S", 0, "Status (0=Todo, 1=In progress, 2=Done)")
	return cmd
}

func kanbanSubtaskUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <task_id> <subtask_id>",
		Short: "Update a subtask",
		Long:  "Modify an existing subtask",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("task ID and subtask ID are required")
			}
			taskId, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}
			subtaskId, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid subtask ID: %w", err)
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			req := client.KanbanUpdateSubtaskRequest{}
			if title, _ := cmd.Flags().GetString("title"); title != "" {
				req.Title = title
			}
			if user, _ := cmd.Flags().GetInt("user"); user >= 0 {
				req.UserID = user
			}
			if te, _ := cmd.Flags().GetInt("time-estimated"); te >= 0 {
				req.TimeEstimated = te
			}
			if ts, _ := cmd.Flags().GetInt("time-spent"); ts >= 0 {
				req.TimeSpent = ts
			}
			if st, _ := cmd.Flags().GetInt("status"); st >= 0 {
				req.Status = st
			}

			result, err := c.Kanban.UpdateSubtask(cmd.Context(), taskId, subtaskId, req)
			if err != nil {
				return fmt.Errorf("update subtask: %w", err)
			}

			if result.Success {
				_, _ = fmt.Printf("Subtask updated: %d\n", subtaskId)
			} else {
				_, _ = fmt.Println("Failed to update subtask")
			}
			return nil
		},
	}
	cmd.Flags().StringP("title", "t", "", "New title")
	cmd.Flags().IntP("user", "u", -1, "User ID to assign (-1 to unassign)")
	cmd.Flags().IntP("time-estimated", "e", -1, "Estimated time (minutes, -1 to clear)")
	cmd.Flags().IntP("time-spent", "s", -1, "Time spent (minutes, -1 to clear)")
	cmd.Flags().IntP("status", "S", -1, "Status (0=Todo, 1=In progress, 2=Done, -1 to leave unchanged)")
	return cmd
}

func kanbanSubtaskDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <task_id> <subtask_id>",
		Short: "Delete a subtask",
		Long:  "Remove a subtask by ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("task ID and subtask ID are required")
			}
			taskId, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}
			subtaskId, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid subtask ID: %w", err)
			}

			yes, _ := cmd.Flags().GetBool("yes")
			if !yes {
				_, _ = fmt.Printf("Delete subtask %d? [y/N]: ", subtaskId)
				var response string
				if _, err := fmt.Scanln(&response); err != nil {
					return fmt.Errorf("read confirmation: %w", err)
				}
				if response != "y" && response != "Y" {
					_, _ = fmt.Println("Cancelled")
					return nil
				}
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			result, err := c.Kanban.RemoveSubtask(cmd.Context(), taskId, subtaskId)
			if err != nil {
				return fmt.Errorf("delete subtask: %w", err)
			}

			if result.Success {
				_, _ = fmt.Printf("Subtask deleted: %d\n", subtaskId)
			} else {
				_, _ = fmt.Println("Failed to delete subtask")
			}
			return nil
		},
	}
	cmd.Flags().BoolP("yes", "y", false, "Skip confirmation")
	return cmd
}

func kanbanSubtaskTimerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "timer",
		Short: "Manage subtask timer",
		Long:  "Check, start, stop timer and get time spent for subtasks",
	}
	cmd.AddCommand(
		kanbanSubtaskTimerCheckCommand(),
		kanbanSubtaskTimerStartCommand(),
		kanbanSubtaskTimerStopCommand(),
		kanbanSubtaskTimerSpentCommand(),
	)
	return cmd
}

func kanbanSubtaskTimerCheckCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check <task_id> <subtask_id>",
		Short: "Check if timer is active",
		Long:  "Check if a timer is started for the given subtask and user",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("task ID and subtask ID are required")
			}
			taskId, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}
			subtaskId, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid subtask ID: %w", err)
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			userId, _ := cmd.Flags().GetInt("user")
			result, err := c.Kanban.HasSubtaskTimer(cmd.Context(), taskId, subtaskId, userId)
			if err != nil {
				return fmt.Errorf("check subtask timer: %w", err)
			}

			if result.Result {
				_, _ = fmt.Println("Timer is active")
			} else {
				_, _ = fmt.Println("Timer is not active")
			}
			return nil
		},
	}
	cmd.Flags().IntP("user", "u", 0, "User ID")
	return cmd
}

func kanbanSubtaskTimerStartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start <task_id> <subtask_id>",
		Short: "Start subtask timer",
		Long:  "Start subtask timer for a user",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("task ID and subtask ID are required")
			}
			taskId, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}
			subtaskId, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid subtask ID: %w", err)
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			userId, _ := cmd.Flags().GetInt("user")
			result, err := c.Kanban.SetSubtaskStartTime(cmd.Context(), taskId, subtaskId, userId)
			if err != nil {
				return fmt.Errorf("start subtask timer: %w", err)
			}

			if result.Result {
				_, _ = fmt.Println("Timer started successfully")
			} else {
				_, _ = fmt.Println("Failed to start timer")
			}
			return nil
		},
	}
	cmd.Flags().IntP("user", "u", 0, "User ID")
	return cmd
}

func kanbanSubtaskTimerStopCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop <task_id> <subtask_id>",
		Short: "Stop subtask timer",
		Long:  "Stop subtask timer for a user",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("task ID and subtask ID are required")
			}
			taskId, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}
			subtaskId, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid subtask ID: %w", err)
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			userId, _ := cmd.Flags().GetInt("user")
			result, err := c.Kanban.SetSubtaskEndTime(cmd.Context(), taskId, subtaskId, userId)
			if err != nil {
				return fmt.Errorf("stop subtask timer: %w", err)
			}

			if result.Result {
				_, _ = fmt.Println("Timer stopped successfully")
			} else {
				_, _ = fmt.Println("Failed to stop timer")
			}
			return nil
		},
	}
	cmd.Flags().IntP("user", "u", 0, "User ID")
	return cmd
}

func kanbanSubtaskTimerSpentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spent <task_id> <subtask_id>",
		Short: "Get time spent",
		Long:  "Get time spent on a subtask for a user (in hours)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("task ID and subtask ID are required")
			}
			taskId, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}
			subtaskId, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid subtask ID: %w", err)
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			userId, _ := cmd.Flags().GetInt("user")
			result, err := c.Kanban.GetSubtaskTimeSpent(cmd.Context(), taskId, subtaskId, userId)
			if err != nil {
				return fmt.Errorf("get subtask time spent: %w", err)
			}

			_, _ = fmt.Printf("Time spent: %.2f hours\n", result.Result)
			return nil
		},
	}
	cmd.Flags().IntP("user", "u", 0, "User ID")
	return cmd
}

func kanbanTagCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tag",
		Short: "Manage kanban tags",
		Long:  "Create, update, list and manage kanban tags",
	}
	cmd.AddCommand(
		kanbanTagListCommand(),
		kanbanTagCreateCommand(),
		kanbanTagUpdateCommand(),
		kanbanTagDeleteCommand(),
		kanbanTagTaskCommand(),
	)
	return cmd
}

func kanbanTagListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all tags",
		Long:  "Display kanban tags",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			projectId, _ := cmd.Flags().GetInt("project")
			var tags []client.KanbanTag
			if projectId > 0 {
				tags, err = c.Kanban.ListTagsByProject(cmd.Context(), projectId)
			} else {
				tags, err = c.Kanban.ListTags(cmd.Context())
			}
			if err != nil {
				return fmt.Errorf("list tags: %w", err)
			}

			if len(tags) == 0 {
				_, _ = fmt.Println("No tags found")
				return nil
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				data, err := sonic.MarshalIndent(tags, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal tags: %w", err)
				}
				_, _ = fmt.Println(string(data))
			} else {
				_, _ = fmt.Printf("%-8s %-30s %-10s\n", "ID", "NAME", "PROJECT")
				_, _ = fmt.Println(strings.Repeat("-", 50))
				for _, t := range tags {
					name := t.Name
					if len(name) > 28 {
						name = name[:25] + "..."
					}
					_, _ = fmt.Printf("%-8s %-30s %-10s\n", t.ID, name, t.ProjectID)
				}
			}

			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	cmd.Flags().IntP("project", "p", 0, "Project ID (if specified, list tags for this project)")
	return cmd
}

func kanbanTagCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new tag",
		Long:  "Add a new tag to the kanban board",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			name, _ := cmd.Flags().GetString("name")
			project, _ := cmd.Flags().GetInt("project")
			color, _ := cmd.Flags().GetString("color")

			req := client.KanbanCreateTagRequest{
				ProjectID: project,
				Name:      name,
				ColorID:   color,
			}

			result, err := c.Kanban.CreateTag(cmd.Context(), req)
			if err != nil {
				return fmt.Errorf("create tag: %w", err)
			}

			_, _ = fmt.Printf("Tag created: ID=%d\n", result.ID)
			return nil
		},
	}
	cmd.Flags().StringP("name", "n", "", "Tag name")
	_ = cmd.MarkFlagRequired("name")
	cmd.Flags().IntP("project", "p", 1, "Project ID")
	cmd.Flags().StringP("color", "c", "", "Color ID")
	return cmd
}

func kanbanTagUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a tag",
		Long:  "Modify an existing tag",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("tag ID is required")
			}
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid tag ID: %w", err)
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			name, _ := cmd.Flags().GetString("name")
			color, _ := cmd.Flags().GetString("color")

			req := client.KanbanUpdateTagRequest{
				Name:    name,
				ColorID: color,
			}

			_, err = c.Kanban.UpdateTag(cmd.Context(), id, req)
			if err != nil {
				return fmt.Errorf("update tag: %w", err)
			}

			_, _ = fmt.Printf("Tag updated: %d\n", id)
			return nil
		},
	}
	cmd.Flags().StringP("name", "n", "", "New tag name")
	_ = cmd.MarkFlagRequired("name")
	cmd.Flags().StringP("color", "c", "", "Color ID")
	return cmd
}

func kanbanTagDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a tag",
		Long:  "Remove a tag by ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("tag ID is required")
			}
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid tag ID: %w", err)
			}

			yes, _ := cmd.Flags().GetBool("yes")
			if !yes {
				_, _ = fmt.Printf("Delete tag %d? [y/N]: ", id)
				var response string
				if _, err := fmt.Scanln(&response); err != nil {
					return fmt.Errorf("read confirmation: %w", err)
				}
				if response != "y" && response != "Y" {
					_, _ = fmt.Println("Cancelled")
					return nil
				}
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			_, err = c.Kanban.RemoveTag(cmd.Context(), id)
			if err != nil {
				return fmt.Errorf("delete tag: %w", err)
			}

			_, _ = fmt.Printf("Tag deleted: %d\n", id)
			return nil
		},
	}
	cmd.Flags().BoolP("yes", "y", false, "Skip confirmation")
	return cmd
}

func kanbanTagTaskCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Manage task tags",
		Long:  "Get or set tags for a task",
	}
	cmd.AddCommand(
		kanbanTagTaskGetCommand(),
		kanbanTagTaskSetCommand(),
	)
	return cmd
}

func kanbanTagTaskGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <task_id>",
		Short: "Get tags for a task",
		Long:  "Display tags assigned to a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("task ID is required")
			}
			taskId, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			tags, err := c.Kanban.GetTaskTags(cmd.Context(), taskId)
			if err != nil {
				return fmt.Errorf("get task tags: %w", err)
			}

			if len(tags) == 0 {
				_, _ = fmt.Println("No tags assigned to this task")
				return nil
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				data, err := sonic.MarshalIndent(tags, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal tags: %w", err)
				}
				_, _ = fmt.Println(string(data))
			} else {
				_, _ = fmt.Printf("%-8s %-30s\n", "ID", "NAME")
				_, _ = fmt.Println(strings.Repeat("-", 40))
				for id, name := range tags {
					_, _ = fmt.Printf("%-8s %-30s\n", id, name)
				}
			}

			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (json, table)")
	return cmd
}

func kanbanTagTaskSetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <task_id>",
		Short: "Set tags for a task",
		Long:  "Assign tags to a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("task ID is required")
			}
			taskId, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid task ID: %w", err)
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			project, _ := cmd.Flags().GetInt("project")
			tags, _ := cmd.Flags().GetStringSlice("tags")

			req := client.KanbanSetTaskTagsRequest{
				ProjectID: project,
				Tags:      tags,
			}

			result, err := c.Kanban.SetTaskTags(cmd.Context(), taskId, req)
			if err != nil {
				return fmt.Errorf("set task tags: %w", err)
			}

			if result.Success {
				_, _ = fmt.Println("Task tags updated successfully")
			} else {
				_, _ = fmt.Println("Failed to update task tags")
			}

			return nil
		},
	}
	cmd.Flags().IntP("project", "p", 0, "Project ID")
	_ = cmd.MarkFlagRequired("project")
	cmd.Flags().StringSliceP("tags", "t", nil, "Tag names (can be specified multiple times)")
	_ = cmd.MarkFlagRequired("tags")
	return cmd
}

func kanbanCardCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "card",
		Short: "Work with kanban cards (alias for task operations)",
		Long:  "Manage cards within kanban boards via server API",
	}
	cmd.AddCommand(
		kanbanCardAddCommand(),
		kanbanCardMoveCommand(),
		kanbanCardDeleteCommand(),
	)
	return cmd
}

func kanbanCardAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a card to a kanban board",
		Long:  "Create a new task in the specified column",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			title, _ := cmd.Flags().GetString("title")
			description, _ := cmd.Flags().GetString("description")
			project, _ := cmd.Flags().GetInt("project")
			column, _ := cmd.Flags().GetInt("column")

			req := client.KanbanCreateRequest{
				Title:       title,
				Description: description,
				ProjectID:   project,
				ColumnID:    column,
			}

			result, err := c.Kanban.Create(cmd.Context(), req)
			if err != nil {
				return fmt.Errorf("create card: %w", err)
			}

			_, _ = fmt.Printf("Card created: ID=%d\n", result.ID)
			return nil
		},
	}
	cmd.Flags().StringP("title", "t", "", "Card title")
	_ = cmd.MarkFlagRequired("title")
	cmd.Flags().StringP("description", "d", "", "Card description")
	cmd.Flags().IntP("project", "p", 1, "Project ID")
	cmd.Flags().IntP("column", "c", 0, "Column ID")
	return cmd
}

func kanbanCardMoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "move <card-id>",
		Short: "Move a card to another column",
		Long:  "Move a task to a different column",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("card ID is required")
			}
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid card ID: %w", err)
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			column, _ := cmd.Flags().GetInt("column")
			position, _ := cmd.Flags().GetInt("position")
			project, _ := cmd.Flags().GetInt("project")

			req := client.KanbanMoveRequest{
				ColumnID:  column,
				Position:  position,
				ProjectID: project,
			}

			_, err = c.Kanban.Move(cmd.Context(), id, req)
			if err != nil {
				return fmt.Errorf("move card: %w", err)
			}

			_, _ = fmt.Printf("Card moved: %d -> column %d\n", id, req.ColumnID)
			return nil
		},
	}
	cmd.Flags().IntP("column", "c", 0, "Destination column ID")
	_ = cmd.MarkFlagRequired("column")
	cmd.Flags().IntP("position", "p", 0, "Position in column (0 = first)")
	cmd.Flags().IntP("project", "r", 1, "Project ID")
	return cmd
}

func kanbanCardDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <card-id>",
		Short: "Delete a card from a kanban board",
		Long:  "Close a task by ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("card ID is required")
			}
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid card ID: %w", err)
			}

			yes, _ := cmd.Flags().GetBool("yes")
			if !yes {
				_, _ = fmt.Printf("Close card %d? [y/N]: ", id)
				var response string
				if _, err := fmt.Scanln(&response); err != nil {
					return fmt.Errorf("read confirmation: %w", err)
				}
				if response != "y" && response != "Y" {
					_, _ = fmt.Println("Cancelled")
					return nil
				}
			}

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			_, err = c.Kanban.Close(cmd.Context(), id)
			if err != nil {
				return fmt.Errorf("close card: %w", err)
			}

			_, _ = fmt.Printf("Card closed: %d\n", id)
			return nil
		},
	}
	cmd.Flags().BoolP("yes", "y", false, "Skip confirmation")
	return cmd
}

func kanbanColumnCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "column",
		Short: "Work with kanban columns",
		Long:  "Manage columns within kanban boards",
	}
	cmd.AddCommand(kanbanColumnListCommand())
	return cmd
}

func kanbanColumnListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List columns in a project",
		Long:  "Display all columns in the specified project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			projectId, _ := cmd.Flags().GetInt("project")

			columns, err := c.Kanban.ListColumns(cmd.Context(), projectId)
			if err != nil {
				return fmt.Errorf("list columns: %w", err)
			}

			if len(columns) == 0 {
				_, _ = fmt.Println("No columns found")
				return nil
			}

			_, _ = fmt.Printf("%-8s %-20s\n", "ID", "TITLE")
			for _, col := range columns {
				_, _ = fmt.Printf("%-8d %-20s\n", col.ID, col.Title)
			}

			return nil
		},
	}
	cmd.Flags().IntP("project", "p", 1, "Project ID")
	return cmd
}
