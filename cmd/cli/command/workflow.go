package command

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/cmd/cli/utils"
)

func WorkflowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workflow",
		Short: "Manage workflows",
		Long:  "Run local workflow YAML files with capability, docker, shell, and machine actions.",
	}
	cmd.AddCommand(workflowRunCommand())
	return cmd
}

func workflowRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <file.yaml>",
		Short: "Run a workflow YAML file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("workflow file path is required")
			}
			filePath := args[0]

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			result, err := c.Workflow.RunFile(cmd.Context(), filePath)
			if err != nil {
				return fmt.Errorf("run workflow: %w", err)
			}

			_, _ = fmt.Printf("Workflow %s completed: %s\n", filePath, result.Message)
			return nil
		},
	}
	return cmd
}
