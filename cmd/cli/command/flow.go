package command

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/cmd/cli/utils"
	"github.com/flowline-io/flowbot/pkg/client"
)

// FlowCommand returns the flow parent command.
func FlowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "flow",
		Short: "Inspect live flow topology",
	}
	cmd.AddCommand(flowGraphCommand())
	return cmd
}

func flowGraphCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "graph",
		Short: "Aggregated app edges from resource_links",
		Long:  "Default window is 7 days. Needs lineage:read.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			includeStatic, _ := cmd.Flags().GetBool("include-static")
			out, err := c.Flow.Graph(cmd.Context(), client.FlowGraphOptions{
				Since:         flagString(cmd, "since"),
				Until:         flagString(cmd, "until"),
				App:           flagString(cmd, "app"),
				IncludeStatic: includeStatic,
			})
			if err != nil {
				return fmt.Errorf("flow graph: %w", err)
			}
			if IsJSON(cmd) {
				return render(cmd, out)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "window %s .. %s  nodes=%d edges=%d\n",
				out.Window.Since.Format("2006-01-02"), out.Window.Until.Format("2006-01-02"),
				len(out.Nodes), len(out.Edges))
			for _, e := range out.Edges {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s -> %s via %s/%s ok=%d failed=%d\n",
					e.Source, e.Target, e.Via.Kind, e.Via.Name, e.CountOK, e.CountFailed)
			}
			return nil
		},
	}
	cmd.Flags().String("since", "", "RFC3339 lower bound (default 7d lookback)")
	cmd.Flags().String("until", "", "RFC3339 upper bound (default now)")
	cmd.Flags().String("app", "", "Optional app filter")
	cmd.Flags().Bool("include-static", true, "Accepted for compatibility; static edges ignored until published")
	return cmd
}
