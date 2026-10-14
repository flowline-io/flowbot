package fireflyiii

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/capability"
)

// CreateTransactionInput holds parameters for creating a finance transaction.
type CreateTransactionInput struct {
	Type            string
	Date            string
	Amount          string
	Description     string
	SourceID        string
	SourceName      string
	DestinationID   string
	DestinationName string
	CategoryName    string
	BudgetID        string
	BudgetName      string
	BillID          string
	BillName        string
	Notes           string
	Tags            []string
}

// UpdateTransactionInput holds partial updates for a transaction group.
// Pointer fields mean "set this value"; nil means leave unchanged.
type UpdateTransactionInput struct {
	ID              string
	Type            *string
	Date            *string
	Amount          *string
	Description     *string
	SourceID        *string
	SourceName      *string
	DestinationID   *string
	DestinationName *string
	CategoryName    *string
	BudgetID        *string
	BudgetName      *string
	BillID          *string
	BillName        *string
	Notes           *string
	Tags            *[]string
}

// TransactionListQuery holds filters and pagination for listing transactions.
type TransactionListQuery struct {
	Page      capability.PageRequest
	Type      string
	Start     string
	End       string
	AccountID string
}

// AccountListQuery holds filters and pagination for listing accounts.
type AccountListQuery struct {
	Page capability.PageRequest
	Type string
}

// ResourceListQuery holds pagination for simple resource lists.
type ResourceListQuery struct {
	Page capability.PageRequest
}

// Service defines the fireflyiii finance capability contract.
type Service interface {
	CreateTransaction(ctx context.Context, in CreateTransactionInput) (*capability.Transaction, error)
	ListTransactions(ctx context.Context, q TransactionListQuery) (*capability.ListResult[capability.Transaction], error)
	GetTransaction(ctx context.Context, id string) (*capability.Transaction, error)
	UpdateTransaction(ctx context.Context, in UpdateTransactionInput) (*capability.Transaction, error)
	DeleteTransaction(ctx context.Context, id string) error
	ListAccounts(ctx context.Context, q AccountListQuery) (*capability.ListResult[capability.FinanceAccount], error)
	ListCategories(ctx context.Context, q ResourceListQuery) (*capability.ListResult[capability.FinanceCategory], error)
	ListBudgets(ctx context.Context, q ResourceListQuery) (*capability.ListResult[capability.FinanceBudget], error)
	ListBills(ctx context.Context, q ResourceListQuery) (*capability.ListResult[capability.FinanceBill], error)
	ListTags(ctx context.Context, q ResourceListQuery) (*capability.ListResult[capability.FinanceTag], error)
	About(ctx context.Context) (*capability.FinanceAbout, error)
	CurrentUser(ctx context.Context) (*capability.FinanceUser, error)
	HealthCheck(ctx context.Context) (bool, error)
}
