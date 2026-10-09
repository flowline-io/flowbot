package command

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/cmd/cli/utils"
)

// BlueprintCommand returns the CLI command for pipeline blueprints.
func BlueprintCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "blueprint",
		Short: "Manage pipeline blueprints",
		Long:  "List, import, instantiate, and update parameterized pipeline templates.",
	}
	cmd.AddCommand(
		blueprintListCommand(),
		blueprintExportCommand(),
		blueprintImportCommand(),
		blueprintReplaceCommand(),
		blueprintDeleteCommand(),
		blueprintInstantiateCommand(),
		blueprintTakeControlCommand(),
		blueprintUpdateCommand(),
	)
	return cmd
}

func blueprintListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List blueprint templates",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			result, err := c.Blueprint.List(cmd.Context())
			if err != nil {
				return fmt.Errorf("list blueprints: %w", err)
			}
			if len(result.Blueprints) == 0 {
				return PrintEmptyList(cmd, "No blueprints")
			}
			output, err := cmd.Flags().GetString("output")
			if err != nil {
				return err
			}
			if output == "json" {
				return PrintJSON(result.Blueprints)
			}
			_, _ = fmt.Printf("%-10s %-32s %-8s %s\n", "SOURCE", "ID", "MISSING", "TITLE")
			for _, b := range result.Blueprints {
				missing := "no"
				if len(b.Missing) > 0 {
					missing = strings.Join(b.Missing, ",")
				}
				_, _ = fmt.Printf("%-10s %-32s %-8s %s\n", b.Source, b.ID, missing, b.Title)
			}
			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func blueprintExportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export <source> <id>",
		Short: "Export blueprint YAML",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			result, err := c.Blueprint.Export(cmd.Context(), args[0], args[1])
			if err != nil {
				return fmt.Errorf("export blueprint: %w", err)
			}
			_, _ = fmt.Fprint(cmd.OutOrStdout(), result.YAML)
			return nil
		},
	}
	return cmd
}

func blueprintImportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import a blueprint YAML file into the user library",
		RunE: func(cmd *cobra.Command, _ []string) error {
			filePath, err := cmd.Flags().GetString("file")
			if err != nil {
				return err
			}
			data, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("read blueprint file: %w", err)
			}
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			result, err := c.Blueprint.Import(cmd.Context(), data)
			if err != nil {
				return fmt.Errorf("import blueprint: %w", err)
			}
			_, _ = fmt.Printf("Imported blueprint %s (%s)\n", result.ID, result.Hash)
			return nil
		},
	}
	cmd.Flags().String("file", "", "Path to blueprint YAML file")
	_ = cmd.MarkFlagRequired("file")
	return cmd
}

func blueprintReplaceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "replace <id>",
		Short: "Replace an imported blueprint YAML",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, err := cmd.Flags().GetString("file")
			if err != nil {
				return err
			}
			data, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("read blueprint file: %w", err)
			}
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			result, err := c.Blueprint.Replace(cmd.Context(), args[0], data)
			if err != nil {
				return fmt.Errorf("replace blueprint: %w", err)
			}
			_, _ = fmt.Printf("Replaced blueprint %s (%s)\n", result.ID, result.Hash)
			return nil
		},
	}
	cmd.Flags().String("file", "", "Path to blueprint YAML file")
	_ = cmd.MarkFlagRequired("file")
	return cmd
}

func blueprintDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an imported blueprint",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			if err := c.Blueprint.Delete(cmd.Context(), args[0]); err != nil {
				return fmt.Errorf("delete blueprint: %w", err)
			}
			_, _ = fmt.Printf("Deleted blueprint %s\n", args[0])
			return nil
		},
	}
}

func blueprintInstantiateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "instantiate <source> <id>",
		Short: "Create a pipeline from a blueprint",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, err := cmd.Flags().GetString("name")
			if err != nil {
				return err
			}
			enable, err := cmd.Flags().GetBool("enable")
			if err != nil {
				return err
			}
			inputsJSON, err := cmd.Flags().GetString("inputs")
			if err != nil {
				return err
			}
			inputs := map[string]any{}
			if strings.TrimSpace(inputsJSON) != "" {
				if err := sonic.Unmarshal([]byte(inputsJSON), &inputs); err != nil {
					return errors.New("inputs must be a JSON object")
				}
			}
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			result, err := c.Blueprint.Instantiate(cmd.Context(), args[0], args[1], name, inputs, enable)
			if err != nil {
				return fmt.Errorf("instantiate blueprint: %w", err)
			}
			_, _ = fmt.Printf("Created pipeline %s (id=%d version=%d)\n", result.Name, result.ID, result.Version)
			return nil
		},
	}
	cmd.Flags().String("name", "", "Pipeline name")
	cmd.Flags().Bool("enable", false, "Enable the pipeline after create")
	cmd.Flags().String("inputs", "{}", "JSON object of blueprint input values")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func blueprintTakeControlCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "take-control <pipeline>",
		Short: "Detach a pipeline from its blueprint",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			if err := c.Blueprint.TakeControl(cmd.Context(), args[0]); err != nil {
				return fmt.Errorf("take control: %w", err)
			}
			_, _ = fmt.Printf("Pipeline %s is now a normal definition\n", args[0])
			return nil
		},
	}
}

func blueprintUpdateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "update <pipeline>",
		Short: "Apply the latest catalog template to a linked pipeline",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			result, err := c.Blueprint.ConfirmUpdate(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("update pipeline from blueprint: %w", err)
			}
			_, _ = fmt.Printf("Updated pipeline %s (version=%d)\n", result.Name, result.Version)
			return nil
		},
	}
}
