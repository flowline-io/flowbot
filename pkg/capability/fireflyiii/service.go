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
	Notes           string
}

// Service defines the fireflyiii finance capability contract.
type Service interface {
	CreateTransaction(ctx context.Context, in CreateTransactionInput) (*capability.Transaction, error)
	About(ctx context.Context) (*capability.FinanceAbout, error)
	CurrentUser(ctx context.Context) (*capability.FinanceUser, error)
	HealthCheck(ctx context.Context) (bool, error)
}
