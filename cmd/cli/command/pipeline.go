package command

import (
	"fmt"
	"os"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/cmd/cli/utils"
)

// PipelineCommand returns the root CLI command for pipeline management.
func PipelineCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pipeline",
		Short: "Manage pipelines",
		Long:  "Apply, list, get, export, delete, and run database-backed pipelines.",
	}
	cmd.AddCommand(
		pipelineApplyCommand(),
		pipelineListCommand(),
		pipelineGetCommand(),
		pipelineExportCommand(),
		pipelineDeleteCommand(),
		pipelineRunCommand(),
		pipelineRunsCommand(),
	)
	return cmd
}

func pipelineApplyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply a pipeline YAML definition",
		RunE: func(cmd *cobra.Command, _ []string) error {
			filePath, err := cmd.Flags().GetString("file")
			if err != nil {
				return err
			}
			if strings.TrimSpace(filePath) == "" {
				return fmt.Errorf("--file is required")
			}
			data, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("read pipeline file: %w", err)
			}
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			result, err := c.Pipeline.Apply(cmd.Context(), data)
			if err != nil {
				return fmt.Errorf("apply pipeline: %w", err)
			}
			_, _ = fmt.Printf("Applied pipeline %s (id=%d enabled=%v version=%d)\n",
				result.Name, result.ID, result.Enabled, result.Version)
			return nil
		},
	}
	cmd.Flags().String("file", "", "Path to pipeline YAML file")
	_ = cmd.MarkFlagRequired("file")
	return cmd
}

func pipelineListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List pipelines",
		Long:  "Display published pipelines.",
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
				return PrintEmptyList(cmd, "No pipelines configured")
			}

			output, err := cmd.Flags().GetString("output")
			if err != nil {
				return err
			}
			if output == "json" {
				return PrintJSON(result.Pipelines)
			}
			_, _ = fmt.Printf("%-32s %-10s %s\n", "NAME", "ENABLED", "TRIGGERS")
			_, _ = fmt.Printf("%s\n", strings.Repeat("-", 60))
			for _, p := range result.Pipelines {
				enabled := "no"
				if p.Enabled {
					enabled = "yes"
				}
				_, _ = fmt.Printf("%-32s %-10s %s\n", p.Name, enabled, strings.Join(p.Triggers, ","))
			}

			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func pipelineGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <name>",
		Short: "Get a pipeline definition",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("pipeline name is required")
			}
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			result, err := c.Pipeline.Get(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("get pipeline: %w", err)
			}
			return PrintJSON(result)
		},
	}
	cmd.Flags().StringP("output", "o", "json", "Output format (json)")
	return cmd
}

func pipelineExportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export <name>",
		Short: "Export a pipeline as YAML",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("pipeline name is required")
			}
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			result, err := c.Pipeline.Export(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("export pipeline: %w", err)
			}
			outPath, err := cmd.Flags().GetString("output")
			if err != nil {
				return err
			}
			if outPath != "" && outPath != "json" && outPath != "table" {
				if err := os.WriteFile(outPath, []byte(result.YAML), 0o644); err != nil {
					return fmt.Errorf("write export file: %w", err)
				}
				_, _ = fmt.Printf("Exported pipeline %s to %s\n", args[0], outPath)
				return nil
			}
			_, _ = fmt.Print(result.YAML)
			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "", "Optional file path to write YAML")
	return cmd
}

func pipelineDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a pipeline definition (also deletes run history)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("pipeline name is required")
			}
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			if err := c.Pipeline.Delete(cmd.Context(), args[0]); err != nil {
				return fmt.Errorf("delete pipeline: %w", err)
			}
			_, _ = fmt.Printf("Deleted pipeline %s\n", args[0])
			return nil
		},
	}
	return cmd
}

func pipelineRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <name>",
		Short: "Run a stored pipeline asynchronously",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("pipeline name is required")
			}
			eventRaw, err := cmd.Flags().GetString("event")
			if err != nil {
				return err
			}
			event := map[string]any{}
			if strings.TrimSpace(eventRaw) != "" {
				if err := sonic.Unmarshal([]byte(eventRaw), &event); err != nil {
					return fmt.Errorf("parse --event JSON: %w", err)
				}
			}
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			result, err := c.Pipeline.Run(cmd.Context(), args[0], event)
			if err != nil {
				return fmt.Errorf("run pipeline: %w", err)
			}
			_, _ = fmt.Printf("Pipeline %s started: run_id=%d\n", args[0], result.RunID)
			return nil
		},
	}
	cmd.Flags().String("event", "{}", "JSON object used as synthetic DataEvent payload")
	return cmd
}

func pipelineRunsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "runs <name>",
		Short: "List runs for a pipeline",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("pipeline name is required")
			}
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			result, err := c.Pipeline.Runs(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("list pipeline runs: %w", err)
			}
			if len(result.Runs) == 0 {
				return PrintEmptyList(cmd, "No runs found")
			}
			output, err := cmd.Flags().GetString("output")
			if err != nil {
				return err
			}
			if output == "json" {
				return PrintJSON(result.Runs)
			}
			_, _ = fmt.Printf("%-10s %-12s %-12s\n", "ID", "STATUS", "TRIGGER")
			_, _ = fmt.Printf("%s\n", strings.Repeat("-", 40))
			for _, r := range result.Runs {
				_, _ = fmt.Printf("%-10d %-12d %-12s\n", r.ID, r.Status, r.TriggerSource)
			}
			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}
