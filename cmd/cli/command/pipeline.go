package command

import (
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/cmd/cli/utils"
)

func PipelineCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pipeline",
		Short: "Manage pipelines",
		Long:  "List and run cross-service pipelines.",
	}
	cmd.AddCommand(
		pipelineListCommand(),
		pipelineRunCommand(),
	)
	return cmd
}

func pipelineListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List pipelines",
		Long:  "Display configured pipelines.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			result, err := c.Pipeline.List(cmd.Context())
			if err != nil {
				return fmt.Errorf("list pipelines: %w", err)
			}

			if len(result.Pipelines) == 0 {
				_, _ = fmt.Println("No pipelines configured")
				return nil
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				data, err := sonic.MarshalIndent(result.Pipelines, "", "  ")
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
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func pipelineRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <name>",
		Short: "Run a pipeline",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("pipeline name is required")
			}
			name := args[0]

			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			result, err := c.Pipeline.Run(cmd.Context(), name)
			if err != nil {
				return fmt.Errorf("run pipeline: %w", err)
			}

			_, _ = fmt.Printf("Pipeline %s triggered: %s\n", name, result.Message)
			return nil
		},
	}
	return cmd
}
