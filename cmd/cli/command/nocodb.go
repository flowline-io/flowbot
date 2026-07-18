// Package command implements CLI command definitions.
package command

import (
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/cmd/cli/utils"
	"github.com/flowline-io/flowbot/pkg/client"
)

// NocodbCommand returns the root command for NocoDB.
func NocodbCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nocodb",
		Short: "Work with NocoDB bases, tables, and records",
		Long:  "Manage NocoDB data via Flowbot server",
	}
	cmd.AddCommand(
		nocodbBasesCommand(),
		nocodbTablesCommand(),
		nocodbTableCommand(),
		nocodbRecordsCommand(),
		nocodbHealthCommand(),
	)
	return cmd
}

func nocodbBasesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bases",
		Short: "List bases",
		Long:  "List NocoDB bases visible to the configured API token (first page)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			result, err := c.Nocodb.ListBases(cmd.Context())
			if err != nil {
				return fmt.Errorf("list bases: %w", err)
			}
			items := result.Items
			if len(items) == 0 {
				return PrintEmptyList(cmd, "No bases found")
			}
			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(result)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-24s %s\n", "ID", "TITLE")
			for _, item := range items {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-24s %s\n", item.ID, item.Title)
			}
			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func nocodbTablesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tables",
		Short: "List tables in a base",
		Long:  "List tables belonging to a NocoDB base (first page)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			baseID, _ := cmd.Flags().GetString("base-id")
			result, err := c.Nocodb.ListTables(cmd.Context(), baseID)
			if err != nil {
				return fmt.Errorf("list tables: %w", err)
			}
			items := result.Items
			if len(items) == 0 {
				return PrintEmptyList(cmd, "No tables found")
			}
			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(result)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-24s %s\n", "ID", "TITLE")
			for _, item := range items {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-24s %s\n", item.ID, item.Title)
			}
			return nil
		},
	}
	cmd.Flags().String("base-id", "", "Base ID")
	_ = cmd.MarkFlagRequired("base-id")
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func nocodbTableCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "table",
		Short: "Get table metadata",
		Long:  "Get NocoDB table metadata including columns",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			tableID, _ := cmd.Flags().GetString("table-id")
			item, err := c.Nocodb.GetTable(cmd.Context(), tableID)
			if err != nil {
				return fmt.Errorf("get table: %w", err)
			}
			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(item)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "ID: %s\n", item.ID)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Title: %s\n", item.Title)
			if item.BaseID != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Base: %s\n", item.BaseID)
			}
			if len(item.Columns) > 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Columns:")
				for _, col := range item.Columns {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %-24s %-20s %s\n", col.ID, col.Title, col.Type)
				}
			}
			return nil
		},
	}
	cmd.Flags().String("table-id", "", "Table ID")
	_ = cmd.MarkFlagRequired("table-id")
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func nocodbRecordsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "records",
		Short: "Work with table records",
		Long:  "List, get, create, update, and delete NocoDB records",
	}
	cmd.AddCommand(
		nocodbRecordsListCommand(),
		nocodbRecordsGetCommand(),
		nocodbRecordsCreateCommand(),
		nocodbRecordsUpdateCommand(),
		nocodbRecordsDeleteCommand(),
	)
	return cmd
}

func nocodbRecordsListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List records",
		Long:  "List records in a NocoDB table",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			tableID, _ := cmd.Flags().GetString("table-id")
			limit, _ := cmd.Flags().GetInt("limit")
			offset, _ := cmd.Flags().GetInt("offset")
			where, _ := cmd.Flags().GetString("where")
			sort, _ := cmd.Flags().GetString("sort")
			fields, _ := cmd.Flags().GetString("fields")
			items, err := c.Nocodb.ListRecords(cmd.Context(), tableID, client.NocoListRecordsQuery{
				Limit: limit, Offset: offset, Where: where, Sort: sort, Fields: fields,
			})
			if err != nil {
				return fmt.Errorf("list records: %w", err)
			}
			if len(items.Items) == 0 {
				return PrintEmptyList(cmd, "No records found")
			}
			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(items)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-12s %s\n", "ID", "FIELDS")
			for _, item := range items.Items {
				raw, _ := sonic.Marshal(item.Fields)
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-12s %s\n", item.ID, string(raw))
			}
			return nil
		},
	}
	cmd.Flags().String("table-id", "", "Table ID")
	_ = cmd.MarkFlagRequired("table-id")
	cmd.Flags().Int("limit", 0, "Max records to return")
	cmd.Flags().Int("offset", 0, "Record offset")
	cmd.Flags().String("where", "", "NocoDB where filter")
	cmd.Flags().String("sort", "", "Sort expression")
	cmd.Flags().String("fields", "", "Comma-separated field names")
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func nocodbRecordsGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a record",
		Long:  "Get a single NocoDB record by ID",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			tableID, _ := cmd.Flags().GetString("table-id")
			recordID, _ := cmd.Flags().GetString("record-id")
			item, err := c.Nocodb.GetRecord(cmd.Context(), tableID, recordID)
			if err != nil {
				return fmt.Errorf("get record: %w", err)
			}
			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(item)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "ID: %s\n", item.ID)
			raw, _ := sonic.MarshalIndent(item.Fields, "", "  ")
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Fields:\n%s\n", string(raw))
			return nil
		},
	}
	cmd.Flags().String("table-id", "", "Table ID")
	_ = cmd.MarkFlagRequired("table-id")
	cmd.Flags().String("record-id", "", "Record ID")
	_ = cmd.MarkFlagRequired("record-id")
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func nocodbRecordsCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a record",
		Long:  "Create a NocoDB record with JSON field values",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			tableID, _ := cmd.Flags().GetString("table-id")
			fieldsJSON, _ := cmd.Flags().GetString("fields")
			fields, err := parseNocoFields(fieldsJSON)
			if err != nil {
				return err
			}
			item, err := c.Nocodb.CreateRecord(cmd.Context(), tableID, fields)
			if err != nil {
				return fmt.Errorf("create record: %w", err)
			}
			_, _ = fmt.Printf("Record created: %s\n", item.ID)
			return nil
		},
	}
	cmd.Flags().String("table-id", "", "Table ID")
	_ = cmd.MarkFlagRequired("table-id")
	cmd.Flags().String("fields", "", `JSON object of field values, e.g. {"Name":"Alice"}`)
	_ = cmd.MarkFlagRequired("fields")
	return cmd
}

func nocodbRecordsUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a record",
		Long:  "Update a NocoDB record with JSON field values",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			tableID, _ := cmd.Flags().GetString("table-id")
			recordID, _ := cmd.Flags().GetString("record-id")
			fieldsJSON, _ := cmd.Flags().GetString("fields")
			fields, err := parseNocoFields(fieldsJSON)
			if err != nil {
				return err
			}
			item, err := c.Nocodb.UpdateRecord(cmd.Context(), tableID, recordID, fields)
			if err != nil {
				return fmt.Errorf("update record: %w", err)
			}
			_, _ = fmt.Printf("Record updated: %s\n", item.ID)
			return nil
		},
	}
	cmd.Flags().String("table-id", "", "Table ID")
	_ = cmd.MarkFlagRequired("table-id")
	cmd.Flags().String("record-id", "", "Record ID")
	_ = cmd.MarkFlagRequired("record-id")
	cmd.Flags().String("fields", "", `JSON object of field values, e.g. {"Name":"Bob"}`)
	_ = cmd.MarkFlagRequired("fields")
	return cmd
}

func nocodbRecordsDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a record",
		Long:  "Delete a NocoDB record by ID",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			tableID, _ := cmd.Flags().GetString("table-id")
			recordID, _ := cmd.Flags().GetString("record-id")
			if err := c.Nocodb.DeleteRecord(cmd.Context(), tableID, recordID); err != nil {
				return fmt.Errorf("delete record: %w", err)
			}
			_, _ = fmt.Printf("Record deleted: %s\n", recordID)
			return nil
		},
	}
	cmd.Flags().String("table-id", "", "Table ID")
	_ = cmd.MarkFlagRequired("table-id")
	cmd.Flags().String("record-id", "", "Record ID")
	_ = cmd.MarkFlagRequired("record-id")
	return cmd
}

func nocodbHealthCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "health",
		Short: "Check NocoDB backend health",
		Long:  "Check whether the NocoDB backend is reachable",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			ok, err := c.Nocodb.Health(cmd.Context())
			if err != nil {
				return fmt.Errorf("health: %w", err)
			}
			if ok {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "NocoDB backend is healthy")
			} else {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "NocoDB backend is unhealthy")
			}
			return nil
		},
	}
	return cmd
}

func parseNocoFields(raw string) (map[string]any, error) {
	if raw == "" {
		return nil, fmt.Errorf("fields are required")
	}
	var fields map[string]any
	if err := sonic.Unmarshal([]byte(raw), &fields); err != nil {
		return nil, fmt.Errorf("invalid fields JSON: %w", err)
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf("fields are required")
	}
	return fields, nil
}
