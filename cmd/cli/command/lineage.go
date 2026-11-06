package command

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/cmd/cli/utils"
	"github.com/flowline-io/flowbot/pkg/client"
)

// LineageCommand returns the lineage parent command.
func LineageCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lineage",
		Short: "Trace and search resource lineage",
	}
	cmd.AddCommand(lineageTraceCommand(), lineageSearchCommand())
	return cmd
}

func lineageTraceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trace",
		Short: "Multi-hop lineage from one seed",
		Long:  "Exactly one of --app/--entity-id, --event-id, or --correlation-id is required. Needs lineage:read.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			opts := client.LineageTraceOptions{
				App:           flagString(cmd, "app"),
				EntityID:      flagString(cmd, "entity-id"),
				EventID:       flagString(cmd, "event-id"),
				CorrelationID: flagString(cmd, "correlation-id"),
				Direction:     flagString(cmd, "direction"),
			}
			if cmd.Flags().Changed("depth") {
				opts.Depth, _ = cmd.Flags().GetInt("depth")
			}
			out, err := c.Lineage.Trace(cmd.Context(), opts)
			if err != nil {
				return fmt.Errorf("lineage trace: %w", err)
			}
			if IsJSON(cmd) {
				return render(cmd, out)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "nodes=%d edges=%d truncated=%v\n", len(out.Nodes), len(out.Edges), out.Truncated)
			for _, e := range out.Edges {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s %s -> %s %s via %s/%s (%s)\n",
					e.SourceApp, e.SourceEntityID, e.TargetApp, e.TargetEntityID, e.RunKind, e.ViaName, e.Status)
			}
			return nil
		},
	}
	cmd.Flags().String("app", "", "Seed app (with --entity-id)")
	cmd.Flags().String("entity-id", "", "Seed entity_id (with --app)")
	cmd.Flags().String("event-id", "", "Seed event_id")
	cmd.Flags().String("correlation-id", "", "Seed correlation_id")
	cmd.Flags().String("direction", "up", "Walk direction: up, down, or both")
	cmd.Flags().Int("depth", 10, "Max hop depth (1-10)")
	return cmd
}

func lineageSearchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search resource nodes by substring",
		Long:  "Matches app, capability, and entity_id. Needs lineage:read.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			opts := client.LineageSearchOptions{
				Query:  flagString(cmd, "query"),
				Cursor: flagString(cmd, "cursor"),
			}
			opts.Limit, _ = cmd.Flags().GetInt("limit")
			out, err := c.Lineage.Search(cmd.Context(), opts)
			if err != nil {
				return fmt.Errorf("lineage search: %w", err)
			}
			stop, err := stopAfterList(cmd, "No matching nodes", out, len(out.Items))
			if err != nil || stop {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-16s %-20s %s\n", "APP", "CAPABILITY", "ENTITY_ID")
			for _, item := range out.Items {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-16s %-20s %s\n", item.App, item.Capability, item.EntityID)
			}
			return nil
		},
	}
	cmd.Flags().String("query", "", "Substring to match")
	cmd.Flags().Int("limit", 20, "Page size")
	cmd.Flags().String("cursor", "", "Opaque page cursor")
	_ = cmd.MarkFlagRequired("query")
	return cmd
}

func flagString(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}
