// Package command implements CLI command definitions.
package command

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/cmd/cli/utils"
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/client"
)

// FireflyiiiCommand returns the root command for Firefly III finance.
func FireflyiiiCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fireflyiii",
		Short: "Work with Firefly III finance",
		Long:  "Manage Firefly III transactions and finance lookups via Flowbot server",
	}
	cmd.AddCommand(
		fireflyiiiCreateCommand(),
		fireflyiiiListCommand(),
		fireflyiiiGetCommand(),
		fireflyiiiUpdateCommand(),
		fireflyiiiDeleteCommand(),
		fireflyiiiAccountsCommand(),
		fireflyiiiCategoriesCommand(),
		fireflyiiiBudgetsCommand(),
		fireflyiiiBillsCommand(),
		fireflyiiiTagsCommand(),
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

			req, err := fireflyiiiWriteRequest(cmd)
			if err != nil {
				return err
			}
			tx, err := c.Fireflyiii.CreateTransaction(cmd.Context(), req)
			if err != nil {
				return fmt.Errorf("create transaction: %w", err)
			}

			if IsJSON(cmd) {
				return render(cmd, tx)
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
	addFireflyiiiWriteFlags(cmd, true)
	return cmd
}

func fireflyiiiListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List transactions",
		Long:  "List Firefly III transaction groups",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			limit, _ := cmd.Flags().GetInt("limit")
			cursor, _ := cmd.Flags().GetString("cursor")
			typ, _ := cmd.Flags().GetString("type")
			start, _ := cmd.Flags().GetString("start")
			end, _ := cmd.Flags().GetString("end")
			accountID, _ := cmd.Flags().GetString("account-id")
			result, err := c.Fireflyiii.ListTransactions(cmd.Context(), &client.ListTransactionsQuery{
				Limit: limit, Cursor: cursor, Type: typ, Start: start, End: end, AccountID: accountID,
			})
			if err != nil {
				return fmt.Errorf("list transactions: %w", err)
			}
			if IsJSON(cmd) {
				if result.Items == nil {
					result.Items = []*capability.Transaction{}
				}
				return render(cmd, result)
			}
			if len(result.Items) == 0 {
				return PrintEmptyList(cmd, "No transactions found")
			}
			printFireflyiiiTransactions(result.Items)
			printNextCursor(result.Page.NextCursor)
			return nil
		},
	}
	cmd.Flags().IntP("limit", "n", 20, "Maximum number of transactions")
	cmd.Flags().StringP("cursor", "c", "", "Pagination cursor")
	cmd.Flags().StringP("type", "t", "", "Transaction type filter")
	cmd.Flags().String("start", "", "Start date (YYYY-MM-DD)")
	cmd.Flags().String("end", "", "End date (YYYY-MM-DD)")
	cmd.Flags().String("account-id", "", "Filter by account ID")
	return cmd
}

func fireflyiiiGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get a transaction",
		Long:  "Display a Firefly III transaction group by ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("transaction ID is required")
			}
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			tx, err := c.Fireflyiii.GetTransaction(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("get transaction: %w", err)
			}
			if IsJSON(cmd) {
				return render(cmd, tx)
			}
			printFireflyiiiTransactionDetail(tx)
			return nil
		},
	}
	return cmd
}

func fireflyiiiUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a transaction",
		Long:  "Partially update a Firefly III transaction group (first split only)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("transaction ID is required")
			}
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			fields := fireflyiiiUpdateFields(cmd)
			if len(fields) == 0 {
				return errors.New("at least one field to update is required")
			}
			tx, err := c.Fireflyiii.UpdateTransaction(cmd.Context(), args[0], fields)
			if err != nil {
				return fmt.Errorf("update transaction: %w", err)
			}
			if IsJSON(cmd) {
				return render(cmd, tx)
			}
			_, _ = fmt.Printf("Transaction updated: %s\n", tx.ID)
			return nil
		},
	}
	addFireflyiiiWriteFlags(cmd, false)
	return cmd
}

func fireflyiiiDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a transaction",
		Long:  "Delete a Firefly III transaction group by ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("transaction ID is required")
			}
			id := args[0]
			if err := utils.Confirm(cmd, fmt.Sprintf("Delete transaction %s? [y/N]: ", id)); err != nil {
				return err
			}
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			if err := c.Fireflyiii.DeleteTransaction(cmd.Context(), id); err != nil {
				return fmt.Errorf("delete transaction: %w", err)
			}
			if IsJSON(cmd) {
				return render(cmd, map[string]string{"id": id})
			}
			_, _ = fmt.Printf("Transaction deleted: %s\n", id)
			return nil
		},
	}
	return cmd
}

func fireflyiiiAccountsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accounts",
		Short: "List accounts",
		Long:  "List Firefly III accounts",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			limit, _ := cmd.Flags().GetInt("limit")
			cursor, _ := cmd.Flags().GetString("cursor")
			typ, _ := cmd.Flags().GetString("type")
			result, err := c.Fireflyiii.ListAccounts(cmd.Context(), &client.ListAccountsQuery{
				Limit: limit, Cursor: cursor, Type: typ,
			})
			if err != nil {
				return fmt.Errorf("list accounts: %w", err)
			}
			if IsJSON(cmd) {
				if result.Items == nil {
					result.Items = []*capability.FinanceAccount{}
				}
				return render(cmd, result)
			}
			if len(result.Items) == 0 {
				return PrintEmptyList(cmd, "No accounts found")
			}
			for _, item := range result.Items {
				_, _ = fmt.Printf("%s\t%s\t%s\n", item.ID, item.Type, item.Name)
			}
			printNextCursor(result.Page.NextCursor)
			return nil
		},
	}
	cmd.Flags().IntP("limit", "n", 50, "Maximum number of accounts")
	cmd.Flags().StringP("cursor", "c", "", "Pagination cursor")
	cmd.Flags().StringP("type", "t", "", "Account type filter (asset, expense, revenue, ...)")
	return cmd
}

func fireflyiiiCategoriesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "categories",
		Short: "List categories",
		Long:  "List Firefly III categories",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			result, err := c.Fireflyiii.ListCategories(cmd.Context(), fireflyiiiResourceQuery(cmd))
			if err != nil {
				return fmt.Errorf("list categories: %w", err)
			}
			if IsJSON(cmd) {
				if result.Items == nil {
					result.Items = []*capability.FinanceCategory{}
				}
				return render(cmd, result)
			}
			if len(result.Items) == 0 {
				return PrintEmptyList(cmd, "No categories found")
			}
			for _, item := range result.Items {
				_, _ = fmt.Printf("%s\t%s\n", item.ID, item.Name)
			}
			printNextCursor(result.Page.NextCursor)
			return nil
		},
	}
	addFireflyiiiResourceListFlags(cmd)
	return cmd
}

func fireflyiiiBudgetsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "budgets",
		Short: "List budgets",
		Long:  "List Firefly III budgets",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			result, err := c.Fireflyiii.ListBudgets(cmd.Context(), fireflyiiiResourceQuery(cmd))
			if err != nil {
				return fmt.Errorf("list budgets: %w", err)
			}
			if IsJSON(cmd) {
				if result.Items == nil {
					result.Items = []*capability.FinanceBudget{}
				}
				return render(cmd, result)
			}
			if len(result.Items) == 0 {
				return PrintEmptyList(cmd, "No budgets found")
			}
			for _, item := range result.Items {
				_, _ = fmt.Printf("%s\t%v\t%s\n", item.ID, item.Active, item.Name)
			}
			printNextCursor(result.Page.NextCursor)
			return nil
		},
	}
	addFireflyiiiResourceListFlags(cmd)
	return cmd
}

func fireflyiiiBillsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bills",
		Short: "List bills",
		Long:  "List Firefly III bills",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			result, err := c.Fireflyiii.ListBills(cmd.Context(), fireflyiiiResourceQuery(cmd))
			if err != nil {
				return fmt.Errorf("list bills: %w", err)
			}
			if IsJSON(cmd) {
				if result.Items == nil {
					result.Items = []*capability.FinanceBill{}
				}
				return render(cmd, result)
			}
			if len(result.Items) == 0 {
				return PrintEmptyList(cmd, "No bills found")
			}
			for _, item := range result.Items {
				_, _ = fmt.Printf("%s\t%v\t%s\n", item.ID, item.Active, item.Name)
			}
			printNextCursor(result.Page.NextCursor)
			return nil
		},
	}
	addFireflyiiiResourceListFlags(cmd)
	return cmd
}

func fireflyiiiTagsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tags",
		Short: "List tags",
		Long:  "List Firefly III tags",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := utils.NewClient(cmd)
			if err != nil {
				return err
			}
			result, err := c.Fireflyiii.ListTags(cmd.Context(), fireflyiiiResourceQuery(cmd))
			if err != nil {
				return fmt.Errorf("list tags: %w", err)
			}
			if IsJSON(cmd) {
				if result.Items == nil {
					result.Items = []*capability.FinanceTag{}
				}
				return render(cmd, result)
			}
			if len(result.Items) == 0 {
				return PrintEmptyList(cmd, "No tags found")
			}
			for _, item := range result.Items {
				_, _ = fmt.Printf("%s\t%s\n", item.ID, item.Name)
			}
			printNextCursor(result.Page.NextCursor)
			return nil
		},
	}
	addFireflyiiiResourceListFlags(cmd)
	return cmd
}

func fireflyiiiResourceQuery(cmd *cobra.Command) *client.ListResourcesQuery {
	limit, _ := cmd.Flags().GetInt("limit")
	cursor, _ := cmd.Flags().GetString("cursor")
	return &client.ListResourcesQuery{Limit: limit, Cursor: cursor}
}

func addFireflyiiiResourceListFlags(cmd *cobra.Command) {
	cmd.Flags().IntP("limit", "n", 50, "Maximum number of items")
	cmd.Flags().StringP("cursor", "c", "", "Pagination cursor")
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

			if IsJSON(cmd) {
				return render(cmd, info)
			}
			_, _ = fmt.Printf("Version:     %s\n", info.Version)
			_, _ = fmt.Printf("API version: %s\n", info.APIVersion)
			_, _ = fmt.Printf("PHP version: %s\n", info.PHPVersion)
			_, _ = fmt.Printf("OS:          %s\n", info.OS)
			_, _ = fmt.Printf("Driver:      %s\n", info.Driver)
			return nil
		},
	}
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

			if IsJSON(cmd) {
				return render(cmd, user)
			}
			_, _ = fmt.Printf("ID:    %s\n", user.ID)
			_, _ = fmt.Printf("Email: %s\n", user.Email)
			_, _ = fmt.Printf("Role:  %s\n", user.Role)
			return nil
		},
	}
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

			if IsJSON(cmd) {
				return render(cmd, map[string]bool{"healthy": healthy})
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

func addFireflyiiiWriteFlags(cmd *cobra.Command, requiredCore bool) {
	cmd.Flags().StringP("type", "t", "", "Transaction type (withdrawal, deposit, transfer)")
	cmd.Flags().String("date", "", "Transaction date (YYYY-MM-DD)")
	cmd.Flags().StringP("amount", "a", "", "Transaction amount")
	cmd.Flags().StringP("description", "m", "", "Transaction description")
	cmd.Flags().String("source-id", "", "Source account ID (required if --source-name omitted)")
	cmd.Flags().String("source-name", "", "Source account name (required if --source-id omitted)")
	cmd.Flags().String("destination-id", "", "Destination account ID (required if --destination-name omitted)")
	cmd.Flags().String("destination-name", "", "Destination account name (required if --destination-id omitted)")
	cmd.Flags().String("category", "", "Category name")
	cmd.Flags().String("budget-id", "", "Budget ID")
	cmd.Flags().String("budget-name", "", "Budget name")
	cmd.Flags().String("bill-id", "", "Bill ID")
	cmd.Flags().String("bill-name", "", "Bill name")
	cmd.Flags().String("notes", "", "Notes")
	cmd.Flags().StringArray("tag", nil, "Tag (repeatable)")
	if requiredCore {
		_ = cmd.MarkFlagRequired("type")
		_ = cmd.MarkFlagRequired("date")
		_ = cmd.MarkFlagRequired("amount")
		_ = cmd.MarkFlagRequired("description")
	}
}

func fireflyiiiWriteRequest(cmd *cobra.Command) (*client.CreateTransactionRequest, error) {
	typ, _ := cmd.Flags().GetString("type")
	date, _ := cmd.Flags().GetString("date")
	amount, _ := cmd.Flags().GetString("amount")
	description, _ := cmd.Flags().GetString("description")
	sourceID, _ := cmd.Flags().GetString("source-id")
	sourceName, _ := cmd.Flags().GetString("source-name")
	destinationID, _ := cmd.Flags().GetString("destination-id")
	destinationName, _ := cmd.Flags().GetString("destination-name")
	categoryName, _ := cmd.Flags().GetString("category")
	budgetID, _ := cmd.Flags().GetString("budget-id")
	budgetName, _ := cmd.Flags().GetString("budget-name")
	billID, _ := cmd.Flags().GetString("bill-id")
	billName, _ := cmd.Flags().GetString("bill-name")
	notes, _ := cmd.Flags().GetString("notes")
	tags, _ := cmd.Flags().GetStringArray("tag")
	return &client.CreateTransactionRequest{
		Type:            typ,
		Date:            date,
		Amount:          amount,
		Description:     description,
		SourceID:        sourceID,
		SourceName:      sourceName,
		DestinationID:   destinationID,
		DestinationName: destinationName,
		CategoryName:    categoryName,
		BudgetID:        budgetID,
		BudgetName:      budgetName,
		BillID:          billID,
		BillName:        billName,
		Notes:           notes,
		Tags:            tags,
	}, nil
}

func fireflyiiiUpdateFields(cmd *cobra.Command) client.UpdateTransactionFields {
	fields := client.UpdateTransactionFields{}
	setIfChanged := func(flag, key string) {
		if cmd.Flags().Changed(flag) {
			v, _ := cmd.Flags().GetString(flag)
			fields[key] = v
		}
	}
	setIfChanged("type", "type")
	setIfChanged("date", "date")
	setIfChanged("amount", "amount")
	setIfChanged("description", "description")
	setIfChanged("source-id", "source_id")
	setIfChanged("source-name", "source_name")
	setIfChanged("destination-id", "destination_id")
	setIfChanged("destination-name", "destination_name")
	setIfChanged("category", "category_name")
	setIfChanged("budget-id", "budget_id")
	setIfChanged("budget-name", "budget_name")
	setIfChanged("bill-id", "bill_id")
	setIfChanged("bill-name", "bill_name")
	setIfChanged("notes", "notes")
	if cmd.Flags().Changed("tag") {
		tags, _ := cmd.Flags().GetStringArray("tag")
		if tags == nil {
			tags = []string{}
		}
		fields["tags"] = tags
	}
	return fields
}

func printFireflyiiiTransactions(items []*capability.Transaction) {
	for _, tx := range items {
		_, _ = fmt.Printf("%s\t%s\t%s\t%s\t%s -> %s\n",
			tx.ID, tx.Date, tx.Amount, tx.Description, tx.SourceName, tx.DestinationName)
	}
}

func printFireflyiiiTransactionDetail(tx *capability.Transaction) {
	_, _ = fmt.Printf("ID:          %s\n", tx.ID)
	_, _ = fmt.Printf("Type:        %s\n", tx.Type)
	_, _ = fmt.Printf("Date:        %s\n", tx.Date)
	_, _ = fmt.Printf("Amount:      %s %s\n", tx.Amount, tx.CurrencyCode)
	_, _ = fmt.Printf("Description: %s\n", tx.Description)
	_, _ = fmt.Printf("Source:      %s (%s)\n", tx.SourceName, tx.SourceID)
	_, _ = fmt.Printf("Destination: %s (%s)\n", tx.DestinationName, tx.DestinationID)
	_, _ = fmt.Printf("Category:    %s\n", tx.CategoryName)
	_, _ = fmt.Printf("Budget:      %s\n", tx.BudgetName)
	_, _ = fmt.Printf("Bill:        %s\n", tx.BillName)
	_, _ = fmt.Printf("Tags:        %v\n", tx.Tags)
	_, _ = fmt.Printf("Notes:       %s\n", tx.Notes)
}
