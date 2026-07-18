package command

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/cmd/cli/utils"
	"github.com/flowline-io/flowbot/pkg/client"
)

// DevopsCommand returns the root command for the devops aggregator.
func DevopsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "devops",
		Short: "Work with devops backends",
		Long:  "Query beszel, uptimekuma, traefik, grafana, wakapi, dozzle, and netalertx via Flowbot server",
	}
	cmd.AddCommand(
		devopsStatusCommand(),
		devopsBeszelCommand(),
		devopsUptimekumaCommand(),
		devopsTraefikCommand(),
		devopsGrafanaCommand(),
		devopsWakapiCommand(),
		devopsDozzleCommand(),
		devopsNetalertxCommand(),
	)
	return cmd
}

func devopsStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show configured devops backends",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			st, err := c.Devops.Status(cmd.Context())
			if err != nil {
				return fmt.Errorf("devops status: %w", err)
			}
			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(st)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-12s %s\n", "BACKEND", "CONFIGURED")
			for name, ok := range st.Backends {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-12s %v\n", name, ok)
			}
			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func devopsBeszelCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "beszel",
		Short: "Beszel host monitoring",
	}
	cmd.AddCommand(devopsBeszelSystemsCommand(), devopsBeszelGetCommand())
	return cmd
}

func devopsBeszelSystemsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "systems",
		Short: "List Beszel systems",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			result, err := c.Devops.BeszelListSystems(cmd.Context())
			if err != nil {
				return fmt.Errorf("list systems: %w", err)
			}
			if len(result.Items) == 0 {
				return PrintEmptyList(cmd, "No systems found")
			}
			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(result)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-24s %-16s %s\n", "ID", "NAME", "STATUS")
			for _, item := range result.Items {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-24s %-16s %s\n", item.ID, item.Name, item.Status)
			}
			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func devopsBeszelGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a Beszel system",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			id, _ := cmd.Flags().GetString("id")
			item, err := c.Devops.BeszelGetSystem(cmd.Context(), id)
			if err != nil {
				return fmt.Errorf("get system: %w", err)
			}
			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(item)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "ID: %s\nName: %s\nStatus: %s\nHost: %s\n", item.ID, item.Name, item.Status, item.Host)
			return nil
		},
	}
	cmd.Flags().String("id", "", "System ID")
	_ = cmd.MarkFlagRequired("id")
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func devopsUptimekumaCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uptimekuma",
		Short: "Uptime Kuma metrics",
	}
	cmd.AddCommand(devopsUptimekumaHealthCommand(), devopsUptimekumaMetricsCommand())
	return cmd
}

func devopsUptimekumaHealthCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Check Uptime Kuma health",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			ok, err := c.Devops.UptimekumaHealth(cmd.Context())
			if err != nil {
				return fmt.Errorf("uptimekuma health: %w", err)
			}
			if ok {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Uptime Kuma backend is healthy")
			} else {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Uptime Kuma backend is unhealthy")
			}
			return nil
		},
	}
}

func devopsUptimekumaMetricsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metrics",
		Short: "Summarize Uptime Kuma Prometheus metrics",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			result, err := c.Devops.UptimekumaMetrics(cmd.Context())
			if err != nil {
				return fmt.Errorf("uptimekuma metrics: %w", err)
			}
			if len(result.Items) == 0 {
				return PrintEmptyList(cmd, "No metrics found")
			}
			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(result)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-40s %s\n", "NAME", "SAMPLES")
			for _, item := range result.Items {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-40s %d\n", item.Name, item.Count)
			}
			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func devopsTraefikCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "traefik",
		Short: "Traefik reverse proxy",
	}
	cmd.AddCommand(devopsTraefikOverviewCommand(), devopsTraefikRoutersCommand(), devopsTraefikServicesCommand())
	return cmd
}

func devopsTraefikOverviewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "overview",
		Short: "Show Traefik overview counts",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			ov, err := c.Devops.TraefikOverview(cmd.Context())
			if err != nil {
				return fmt.Errorf("traefik overview: %w", err)
			}
			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(ov)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "HTTP routers: %d\nHTTP services: %d\n", ov.HTTPRouters, ov.HTTPServices)
			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func devopsTraefikRoutersCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "routers",
		Short: "List Traefik HTTP routers",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			result, err := c.Devops.TraefikListRouters(cmd.Context())
			if err != nil {
				return fmt.Errorf("list routers: %w", err)
			}
			if len(result.Items) == 0 {
				return PrintEmptyList(cmd, "No routers found")
			}
			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(result)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-32s %-12s %s\n", "NAME", "STATUS", "RULE")
			for _, item := range result.Items {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-32s %-12s %s\n", item.Name, item.Status, item.Rule)
			}
			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func devopsTraefikServicesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "services",
		Short: "List Traefik HTTP services",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			result, err := c.Devops.TraefikListServices(cmd.Context())
			if err != nil {
				return fmt.Errorf("list services: %w", err)
			}
			if len(result.Items) == 0 {
				return PrintEmptyList(cmd, "No services found")
			}
			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(result)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-32s %-12s %s\n", "NAME", "STATUS", "TYPE")
			for _, item := range result.Items {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-32s %-12s %s\n", item.Name, item.Status, item.Type)
			}
			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func devopsGrafanaCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "grafana",
		Short: "Grafana dashboards, datasources, and observability queries",
	}
	cmd.AddCommand(
		devopsGrafanaHealthCommand(),
		devopsGrafanaDatasourcesCommand(),
		devopsGrafanaDashboardsCommand(),
		devopsGrafanaQueryCommand(),
	)
	return cmd
}

func devopsGrafanaHealthCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "health",
		Short: "Check Grafana health",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			h, err := c.Devops.GrafanaHealth(cmd.Context())
			if err != nil {
				return fmt.Errorf("grafana health: %w", err)
			}
			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(h)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Database: %s\nVersion: %s\n", h.Database, h.Version)
			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func devopsGrafanaDatasourcesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "datasources",
		Short: "List Grafana datasources",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			result, err := c.Devops.GrafanaListDatasources(cmd.Context())
			if err != nil {
				return fmt.Errorf("list datasources: %w", err)
			}
			if len(result.Items) == 0 {
				return PrintEmptyList(cmd, "No datasources found")
			}
			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(result)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-8s %-24s %s\n", "ID", "NAME", "TYPE")
			for _, item := range result.Items {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-8d %-24s %s\n", item.ID, item.Name, item.Type)
			}
			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func devopsGrafanaDashboardsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dashboards",
		Short: "Search Grafana dashboards",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			query, _ := cmd.Flags().GetString("query")
			result, err := c.Devops.GrafanaSearchDashboards(cmd.Context(), query)
			if err != nil {
				return fmt.Errorf("search dashboards: %w", err)
			}
			if len(result.Items) == 0 {
				return PrintEmptyList(cmd, "No dashboards found")
			}
			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(result)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-16s %s\n", "UID", "TITLE")
			for _, item := range result.Items {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-16s %s\n", item.UID, item.Title)
			}
			return nil
		},
	}
	cmd.Flags().String("query", "", "Search query")
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func devopsGrafanaQueryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query prometheus, alloy, loki, tempo, or pyroscope via Grafana",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			backend, _ := cmd.Flags().GetString("backend")
			expr, _ := cmd.Flags().GetString("expr")
			uid, _ := cmd.Flags().GetString("datasource-uid")
			from, _ := cmd.Flags().GetString("from")
			to, _ := cmd.Flags().GetString("to")
			maxLines, _ := cmd.Flags().GetInt("max-lines")
			result, err := c.Devops.GrafanaQuery(cmd.Context(), client.DevopsGrafanaQueryRequest{
				Backend: backend, Expr: expr, DatasourceUID: uid, From: from, To: to, MaxLines: maxLines,
			})
			if err != nil {
				return fmt.Errorf("grafana query: %w", err)
			}
			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(result)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Backend: %s\nDatasource: %s (%s)\nFrames: %d\n",
				result.Backend, result.DatasourceUID, result.DatasourceType, len(result.Frames))
			for _, frame := range result.Frames {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "- %s ref=%s fields=%d\n", frame.Name, frame.RefID, len(frame.Fields))
			}
			return nil
		},
	}
	cmd.Flags().String("backend", "", "prometheus|alloy|loki|tempo|pyroscope")
	_ = cmd.MarkFlagRequired("backend")
	cmd.Flags().String("expr", "", "Query expression (PromQL/LogQL/TraceQL/label selector)")
	_ = cmd.MarkFlagRequired("expr")
	cmd.Flags().String("datasource-uid", "", "Optional Grafana datasource UID")
	cmd.Flags().String("from", "now-1h", "Grafana from time")
	cmd.Flags().String("to", "now", "Grafana to time")
	cmd.Flags().Int("max-lines", 100, "Loki max lines")
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func devopsWakapiCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wakapi",
		Short: "Wakapi coding stats",
	}
	cmd.AddCommand(devopsWakapiSummaryCommand(), devopsWakapiProjectsCommand())
	return cmd
}

func devopsWakapiSummaryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "summary",
		Short: "Show Wakapi activity summary",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			interval, _ := cmd.Flags().GetString("interval")
			s, err := c.Devops.WakapiSummary(cmd.Context(), interval)
			if err != nil {
				return fmt.Errorf("wakapi summary: %w", err)
			}
			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(s)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Total seconds: %d\n", s.TotalSeconds)
			return nil
		},
	}
	cmd.Flags().String("interval", "today", "Summary interval")
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func devopsWakapiProjectsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "projects",
		Short: "List Wakapi projects",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			result, err := c.Devops.WakapiListProjects(cmd.Context())
			if err != nil {
				return fmt.Errorf("list projects: %w", err)
			}
			if len(result.Items) == 0 {
				return PrintEmptyList(cmd, "No projects found")
			}
			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(result)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-24s %s\n", "ID", "NAME")
			for _, item := range result.Items {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-24s %s\n", item.ID, item.Name)
			}
			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func devopsDozzleCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dozzle",
		Short: "Dozzle Docker log viewer",
	}
	cmd.AddCommand(devopsDozzleHealthCommand())
	return cmd
}

func devopsDozzleHealthCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "health",
		Short: "Check Dozzle health",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			info, err := c.Devops.DozzleHealth(cmd.Context())
			if err != nil {
				return fmt.Errorf("dozzle health: %w", err)
			}
			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(info)
			}
			if info.Healthy {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Dozzle is healthy (version %s)\n", info.Version)
			} else {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Dozzle is unhealthy")
			}
			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func devopsNetalertxCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "netalertx",
		Short: "NetAlertX network devices",
	}
	cmd.AddCommand(
		devopsNetalertxHealthCommand(),
		devopsNetalertxDevicesCommand(),
		devopsNetalertxTotalsCommand(),
		devopsNetalertxSearchCommand(),
	)
	return cmd
}

func devopsNetalertxHealthCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Check NetAlertX health",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			ok, err := c.Devops.NetalertxHealth(cmd.Context())
			if err != nil {
				return fmt.Errorf("netalertx health: %w", err)
			}
			if ok {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "NetAlertX backend is healthy")
			} else {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "NetAlertX backend is unhealthy")
			}
			return nil
		},
	}
}

func devopsNetalertxDevicesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "devices",
		Short: "List NetAlertX devices",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			result, err := c.Devops.NetalertxListDevices(cmd.Context())
			if err != nil {
				return fmt.Errorf("list devices: %w", err)
			}
			if len(result.Items) == 0 {
				return PrintEmptyList(cmd, "No devices found")
			}
			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(result)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-20s %-18s %-16s %s\n", "NAME", "MAC", "IP", "STATUS")
			for _, item := range result.Items {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-20s %-18s %-16s %s\n", item.Name, item.MAC, item.IP, item.Status)
			}
			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func devopsNetalertxTotalsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "totals",
		Short: "Show NetAlertX device totals",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			t, err := c.Devops.NetalertxTotals(cmd.Context())
			if err != nil {
				return fmt.Errorf("netalertx totals: %w", err)
			}
			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(t)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "All: %d\nConnected: %d\nFavorites: %d\nNew: %d\nDown: %d\nArchived: %d\n",
				t.All, t.Connected, t.Favorites, t.New, t.Down, t.Archived)
			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func devopsNetalertxSearchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search NetAlertX devices",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			query, _ := cmd.Flags().GetString("query")
			result, err := c.Devops.NetalertxSearchDevices(cmd.Context(), query)
			if err != nil {
				return fmt.Errorf("search devices: %w", err)
			}
			if len(result.Items) == 0 {
				return PrintEmptyList(cmd, "No devices found")
			}
			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(result)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-20s %-18s %-16s %s\n", "NAME", "MAC", "IP", "STATUS")
			for _, item := range result.Items {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-20s %-18s %-16s %s\n", item.Name, item.MAC, item.IP, item.Status)
			}
			return nil
		},
	}
	cmd.Flags().String("query", "", "Search query (MAC, name, or IP)")
	_ = cmd.MarkFlagRequired("query")
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}
