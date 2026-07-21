package command

import (
	"fmt"
	"os"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/cmd/cli/utils"
)

// WorkflowCommand returns the root CLI command for workflow management.
func WorkflowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workflow",
		Short: "Manage workflows",
		Long:  "Apply, list, get, export, delete, and run database-backed workflows.",
	}
	cmd.AddCommand(
		workflowApplyCommand(),
		workflowListCommand(),
		workflowGetCommand(),
		workflowExportCommand(),
		workflowDeleteCommand(),
		workflowRunCommand(),
		workflowRunsCommand(),
	)
	return cmd
}

func workflowApplyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply a workflow YAML definition",
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
				return fmt.Errorf("read workflow file: %w", err)
			}
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			result, err := c.Workflow.Apply(cmd.Context(), data)
			if err != nil {
				return fmt.Errorf("apply workflow: %w", err)
			}
			_, _ = fmt.Printf("Applied workflow %s (id=%d enabled=%v)\n", result.Name, result.ID, result.Enabled)
			return nil
		},
	}
	cmd.Flags().String("file", "", "Path to workflow YAML file")
	_ = cmd.MarkFlagRequired("file")
	return cmd
}

func workflowListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List workflows",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			result, err := c.Workflow.List(cmd.Context())
			if err != nil {
				return fmt.Errorf("list workflows: %w", err)
			}
			if len(result.Workflows) == 0 {
				return PrintEmptyList(cmd, "No workflows configured")
			}
			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(result.Workflows)
			}
			_, _ = fmt.Printf("%-32s %-10s %s\n", "NAME", "ENABLED", "DESCRIBE")
			_, _ = fmt.Printf("%s\n", strings.Repeat("-", 60))
			for _, w := range result.Workflows {
				enabled := "no"
				if w.Enabled {
					enabled = "yes"
				}
				_, _ = fmt.Printf("%-32s %-10s %s\n", w.Name, enabled, w.Describe)
			}
			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func workflowGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <name>",
		Short: "Get a workflow definition",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("workflow name is required")
			}
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			result, err := c.Workflow.Get(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("get workflow: %w", err)
			}
			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(result)
			}
			return PrintJSON(result)
		},
	}
	cmd.Flags().StringP("output", "o", "json", "Output format (json)")
	return cmd
}

func workflowExportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export <name>",
		Short: "Export a workflow as YAML",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("workflow name is required")
			}
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			result, err := c.Workflow.Export(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("export workflow: %w", err)
			}
			outPath, _ := cmd.Flags().GetString("output")
			if outPath != "" && outPath != "json" && outPath != "table" {
				if err := os.WriteFile(outPath, []byte(result.YAML), 0o644); err != nil {
					return fmt.Errorf("write export file: %w", err)
				}
				_, _ = fmt.Printf("Exported workflow %s to %s\n", args[0], outPath)
				return nil
			}
			_, _ = fmt.Print(result.YAML)
			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "", "Optional file path to write YAML")
	return cmd
}

func workflowDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a workflow definition",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("workflow name is required")
			}
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			if err := c.Workflow.Delete(cmd.Context(), args[0]); err != nil {
				return fmt.Errorf("delete workflow: %w", err)
			}
			_, _ = fmt.Printf("Deleted workflow %s\n", args[0])
			return nil
		},
	}
	return cmd
}

func workflowRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <name>",
		Short: "Run a stored workflow asynchronously",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("workflow name is required")
			}
			inputRaw, _ := cmd.Flags().GetString("input")
			input := map[string]any{}
			if strings.TrimSpace(inputRaw) != "" {
				if err := sonic.Unmarshal([]byte(inputRaw), &input); err != nil {
					return fmt.Errorf("parse --input JSON: %w", err)
				}
			}
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			result, err := c.Workflow.Run(cmd.Context(), args[0], input)
			if err != nil {
				return fmt.Errorf("run workflow: %w", err)
			}
			_, _ = fmt.Printf("Workflow %s started: run_id=%d\n", args[0], result.RunID)
			return nil
		},
	}
	cmd.Flags().String("input", "{}", "JSON object of workflow inputs")
	return cmd
}

func workflowRunsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "runs <name>",
		Short: "List runs for a workflow",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("workflow name is required")
			}
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			result, err := c.Workflow.Runs(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("list workflow runs: %w", err)
			}
			if len(result.Runs) == 0 {
				return PrintEmptyList(cmd, "No runs found")
			}
			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(result.Runs)
			}
			_, _ = fmt.Printf("%-10s %-12s %-12s\n", "ID", "STATUS", "TRIGGER")
			_, _ = fmt.Printf("%s\n", strings.Repeat("-", 40))
			for _, r := range result.Runs {
				_, _ = fmt.Printf("%-10d %-12d %-12s\n", r.ID, r.Status, r.TriggerType)
			}
			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}
