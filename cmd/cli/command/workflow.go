package command

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/cmd/cli/utils"
	"github.com/urfave/cli/v3"
)

func WorkflowCommand() *cli.Command {
	return &cli.Command{
		Name:        "workflow",
		Usage:       "Manage workflows",
		Description: "Run local workflow YAML files with capability, docker, shell, and machine actions.",
		Commands: []*cli.Command{
			workflowRunCommand(),
		},
	}
}

func workflowRunCommand() *cli.Command {
	return &cli.Command{
		Name:      "run",
		Usage:     "Run a workflow YAML file",
		ArgsUsage: "<file.yaml>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() == 0 {
				return fmt.Errorf("workflow file path is required")
			}
			filePath := cmd.Args().Get(0)

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			result, err := c.Workflow.RunFile(ctx, filePath)
			if err != nil {
				return fmt.Errorf("run workflow: %w", err)
			}

			_, _ = fmt.Printf("Workflow %s completed: %s\n", filePath, result.Message)
			return nil
		},
	}
}
