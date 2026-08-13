package command

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/cmd/cli/utils"
	pkgfunctions "github.com/flowline-io/flowbot/pkg/functions"
)

// FunctionCommand returns the root CLI command for named function management.
func FunctionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "function",
		Short: "Manage named functions",
		Long:  "Apply, list, get, export, delete, and inspect runs for database-backed named functions.",
	}
	cmd.AddCommand(
		functionApplyCommand(),
		functionListCommand(),
		functionGetCommand(),
		functionExportCommand(),
		functionDeleteCommand(),
		functionRunsCommand(),
	)
	return cmd
}

func functionApplyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply a function directory",
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := cmd.Flags().GetString("dir")
			if err != nil {
				return err
			}
			if strings.TrimSpace(dir) == "" {
				return fmt.Errorf("--dir is required")
			}
			meta, entrypoint, source, err := pkgfunctions.LoadDir(dir)
			if err != nil {
				return err
			}
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			result, err := c.Function.Apply(cmd.Context(), meta, entrypoint, source)
			if err != nil {
				return fmt.Errorf("apply function: %w", err)
			}
			_, _ = fmt.Printf("Applied function %s (id=%d version=%d status=%s)\n",
				result.Name, result.ID, result.Version, result.Status)
			return nil
		},
	}
	cmd.Flags().String("dir", "", "Path to function directory (metadata.yaml + main.py|main.sh|main.go)")
	_ = cmd.MarkFlagRequired("dir")
	return cmd
}

func functionListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List functions",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			result, err := c.Function.List(cmd.Context())
			if err != nil {
				return fmt.Errorf("list functions: %w", err)
			}
			if len(result.Functions) == 0 {
				return PrintEmptyList(cmd, "No functions configured")
			}
			output, err := cmd.Flags().GetString("output")
			if err != nil {
				return err
			}
			if output == "json" {
				return PrintJSON(result.Functions)
			}
			_, _ = fmt.Printf("%-32s %-10s %s\n", "NAME", "VERSION", "STATUS")
			_, _ = fmt.Printf("%s\n", strings.Repeat("-", 54))
			for _, fn := range result.Functions {
				_, _ = fmt.Printf("%-32s %-10d %s\n", fn.Name, fn.Version, fn.Status)
			}
			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func functionGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <name>",
		Short: "Get a function definition",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("function name is required")
			}
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			meta, err := c.Function.Get(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("get function: %w", err)
			}
			return PrintJSON(meta)
		},
	}
	return cmd
}

func functionExportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export <name>",
		Short: "Export a function snapshot",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("function name is required")
			}
			outDir, err := cmd.Flags().GetString("out")
			if err != nil {
				return err
			}
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			bundle, err := c.Function.Export(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("export function: %w", err)
			}
			if strings.TrimSpace(outDir) != "" {
				if err := os.MkdirAll(outDir, 0o755); err != nil {
					return fmt.Errorf("create output dir: %w", err)
				}
				if err := os.WriteFile(filepath.Join(outDir, "metadata.yaml"), []byte(bundle.Metadata), 0o644); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(outDir, bundle.Entrypoint), []byte(bundle.Source), 0o644); err != nil {
					return err
				}
				_, _ = fmt.Printf("Exported function %s v%d to %s\n", bundle.Name, bundle.Version, outDir)
				return nil
			}
			return PrintJSON(bundle)
		},
	}
	cmd.Flags().String("out", "", "Optional directory to write metadata.yaml + entrypoint")
	return cmd
}

func functionDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a function",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("function name is required")
			}
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			if err := c.Function.Delete(cmd.Context(), args[0]); err != nil {
				return fmt.Errorf("delete function: %w", err)
			}
			_, _ = fmt.Printf("Deleted function %s\n", args[0])
			return nil
		},
	}
	return cmd
}

func functionRunsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "runs <name>",
		Short: "List function runs",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("function name is required")
			}
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			result, err := c.Function.Runs(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("list function runs: %w", err)
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
			_, _ = fmt.Printf("%-8s %-8s %-12s %s\n", "ID", "VERSION", "STATUS", "DURATION_MS")
			_, _ = fmt.Printf("%s\n", strings.Repeat("-", 48))
			for _, r := range result.Runs {
				_, _ = fmt.Printf("%-8d %-8d %-12s %d\n", r.ID, r.Version, r.Status, r.DurationMs)
			}
			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}
