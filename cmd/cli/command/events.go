package command

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/cmd/cli/utils"
	"github.com/flowline-io/flowbot/pkg/client"
)

// EventsCommand returns the events parent command.
func EventsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "events",
		Short: "Query the data_events timeline",
	}
	cmd.AddCommand(eventsTimelineCommand())
	return cmd
}

func eventsTimelineCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "timeline",
		Short: "Paged data_events timeline",
		Long:  "Payload bodies are omitted unless --include-payload and the matching service read scope. Needs lineage:read.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			includePayload, _ := cmd.Flags().GetBool("include-payload")
			opts := client.EventsTimelineOptions{
				Since:          flagString(cmd, "since"),
				Until:          flagString(cmd, "until"),
				App:            flagString(cmd, "app"),
				EventType:      flagString(cmd, "event-type"),
				CorrelationID:  flagString(cmd, "correlation-id"),
				Cursor:         flagString(cmd, "cursor"),
				IncludePayload: includePayload,
			}
			opts.Limit, _ = cmd.Flags().GetInt("limit")
			out, err := c.Events.Timeline(cmd.Context(), opts)
			if err != nil {
				return fmt.Errorf("events timeline: %w", err)
			}
			stop, err := stopAfterList(cmd, "No events", out, len(out.Items))
			if err != nil || stop {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-24s %-28s %s\n", "CREATED", "EVENT_TYPE", "EVENT_ID")
			for _, item := range out.Items {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-24s %-28s %s\n",
					item.CreatedAt.Format("2006-01-02T15:04:05Z"), item.EventType, item.EventID)
			}
			return nil
		},
	}
	cmd.Flags().String("since", "", "RFC3339 lower bound (default 7d lookback)")
	cmd.Flags().String("until", "", "RFC3339 upper bound (default now)")
	cmd.Flags().String("app", "", "Optional app filter")
	cmd.Flags().String("event-type", "", "Optional event_type filter")
	cmd.Flags().String("correlation-id", "", "Optional correlation_id filter")
	cmd.Flags().Int("limit", 20, "Page size")
	cmd.Flags().String("cursor", "", "Opaque page cursor")
	cmd.Flags().Bool("include-payload", false, "Include event data when permitted")
	return cmd
}
