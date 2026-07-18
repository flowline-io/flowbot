// Package command implements CLI command definitions.
package command

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/cmd/cli/utils"
	"github.com/flowline-io/flowbot/pkg/client"
)

// FireflyiiiCommand returns the root command for Firefly III finance.
func FireflyiiiCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fireflyiii",
		Short: "Work with Firefly III finance",
		Long:  "Manage Firefly III transactions via Flowbot server",
	}
	cmd.AddCommand(
		fireflyiiiCreateCommand(),
		fireflyiiiAboutCommand(),
		fireflyiiiUserCommand(),
		fireflyiiiHealthCommand(),
	)
	return cmd
}

func fireflyiiiCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a transaction",
		Long:  "Create a new Firefly III transaction (requires source and destination account id or name)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			typ, _ := cmd.Flags().GetString("type")
			date, _ := cmd.Flags().GetString("date")
			amount, _ := cmd.Flags().GetString("amount")
			description, _ := cmd.Flags().GetString("description")
			sourceID, _ := cmd.Flags().GetString("source-id")
			sourceName, _ := cmd.Flags().GetString("source-name")
			destinationID, _ := cmd.Flags().GetString("destination-id")
			destinationName, _ := cmd.Flags().GetString("destination-name")
			categoryName, _ := cmd.Flags().GetString("category")
			notes, _ := cmd.Flags().GetString("notes")

			tx, err := c.Fireflyiii.CreateTransaction(cmd.Context(), &client.CreateTransactionRequest{
				Type:            typ,
				Date:            date,
				Amount:          amount,
				Description:     description,
				SourceID:        sourceID,
				SourceName:      sourceName,
				DestinationID:   destinationID,
				DestinationName: destinationName,
				CategoryName:    categoryName,
				Notes:           notes,
			})
			if err != nil {
				return fmt.Errorf("create transaction: %w", err)
			}

			_, _ = fmt.Printf("Transaction created: %s\n", tx.ID)
			if tx.Description != "" {
				_, _ = fmt.Printf("Description: %s\n", tx.Description)
			}
			if tx.Amount != "" {
				_, _ = fmt.Printf("Amount: %s", tx.Amount)
				if tx.CurrencyCode != "" {
					_, _ = fmt.Printf(" %s", tx.CurrencyCode)
				}
				_, _ = fmt.Println()
			}
			return nil
		},
	}
	cmd.Flags().StringP("type", "t", "", "Transaction type (withdrawal, deposit, transfer)")
	_ = cmd.MarkFlagRequired("type")
	cmd.Flags().String("date", "", "Transaction date (YYYY-MM-DD)")
	_ = cmd.MarkFlagRequired("date")
	cmd.Flags().StringP("amount", "a", "", "Transaction amount")
	_ = cmd.MarkFlagRequired("amount")
	cmd.Flags().StringP("description", "m", "", "Transaction description")
	_ = cmd.MarkFlagRequired("description")
	cmd.Flags().String("source-id", "", "Source account ID (required if --source-name omitted)")
	cmd.Flags().String("source-name", "", "Source account name (required if --source-id omitted)")
	cmd.Flags().String("destination-id", "", "Destination account ID (required if --destination-name omitted)")
	cmd.Flags().String("destination-name", "", "Destination account name (required if --destination-id omitted)")
	cmd.Flags().String("category", "", "Category name")
	cmd.Flags().String("notes", "", "Notes")
	return cmd
}

func fireflyiiiAboutCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "about",
		Short: "Show Firefly III about info",
		Long:  "Display Firefly III instance metadata",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			info, err := c.Fireflyiii.About(cmd.Context())
			if err != nil {
				return fmt.Errorf("get about: %w", err)
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(info)
			}
			_, _ = fmt.Printf("Version:     %s\n", info.Version)
			_, _ = fmt.Printf("API version: %s\n", info.APIVersion)
			_, _ = fmt.Printf("PHP version: %s\n", info.PHPVersion)
			_, _ = fmt.Printf("OS:          %s\n", info.OS)
			_, _ = fmt.Printf("Driver:      %s\n", info.Driver)
			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func fireflyiiiUserCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Show current Firefly III user",
		Long:  "Display the authenticated Firefly III user",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			user, err := c.Fireflyiii.CurrentUser(cmd.Context())
			if err != nil {
				return fmt.Errorf("get current user: %w", err)
			}

			output, _ := cmd.Flags().GetString("output")
			if output == "json" {
				return PrintJSON(user)
			}
			_, _ = fmt.Printf("ID:    %s\n", user.ID)
			_, _ = fmt.Printf("Email: %s\n", user.Email)
			_, _ = fmt.Printf("Role:  %s\n", user.Role)
			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "table", "Output format (table, json)")
	return cmd
}

func fireflyiiiHealthCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "health",
		Short: "Check Firefly III backend health",
		Long:  "Check whether the Firefly III backend is reachable",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}

			healthy, err := c.Fireflyiii.Health(cmd.Context())
			if err != nil {
				return fmt.Errorf("check health: %w", err)
			}

			if healthy {
				_, _ = fmt.Println("Firefly III backend is healthy")
			} else {
				_, _ = fmt.Println("Firefly III backend is NOT healthy")
			}
			return nil
		},
	}
	return cmd
}
