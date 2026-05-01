package finance

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/ability"
)

type CreateTransactionRequest struct {
	Description   string
	Amount        string
	Date          string
	SourceID      string
	DestinationID int
}

type Service interface {
	CreateTransaction(ctx context.Context, req CreateTransactionRequest) (map[string]any, error)
}

type TransactionQuery struct {
	Page ability.PageRequest
}

type TransactionListResult = ability.ListResult[map[string]any]
