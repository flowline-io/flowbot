package command

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/flowline-io/flowbot/cmd/cli/utils"
	"github.com/urfave/cli/v3"
)

func PipelineCommand() *cli.Command {
	return &cli.Command{
		Name:        "pipeline",
		Usage:       "Manage pipelines",
		Description: "List and run cross-service pipelines.",
		Commands: []*cli.Command{
			pipelineListCommand(),
			pipelineRunCommand(),
		},
	}
}

func pipelineListCommand() *cli.Command {
	return &cli.Command{
		Name:        "list",
		Usage:       "List pipelines",
		Description: "Display configured pipelines.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output format (table, json)",
				Value:   "table",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			result, err := c.Pipeline.List(ctx)
			if err != nil {
				return fmt.Errorf("list pipelines: %w", err)
			}

			if len(result.Pipelines) == 0 {
				_, _ = fmt.Println("No pipelines configured")
				return nil
			}

			output := cmd.String("output")
			if output == "json" {
				data, err := json.MarshalIndent(result.Pipelines, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal pipelines: %w", err)
				}
				_, _ = fmt.Println(string(data))
			} else {
				_, _ = fmt.Printf("%-32s %-10s %s\n", "NAME", "ENABLED", "TRIGGER")
				_, _ = fmt.Printf("%s\n", strings.Repeat("-", 60))
				for _, p := range result.Pipelines {
					enabled := "no"
					if p.Enabled {
						enabled = "yes"
					}
					_, _ = fmt.Printf("%-32s %-10s %s\n", p.Name, enabled, p.Trigger.Event)
				}
			}

			return nil
		},
	}
}

func pipelineRunCommand() *cli.Command {
	return &cli.Command{
		Name:      "run",
		Usage:     "Run a pipeline",
		ArgsUsage: "<name>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() == 0 {
				return fmt.Errorf("pipeline name is required")
			}
			name := cmd.Args().Get(0)

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			result, err := c.Pipeline.Run(ctx, name)
			if err != nil {
				return fmt.Errorf("run pipeline: %w", err)
			}

			_, _ = fmt.Printf("Pipeline %s triggered: %s\n", name, result.Message)
			return nil
		},
	}
}
