package fireflyiii

import (
	"context"
	"strconv"

	"github.com/flowline-io/flowbot/pkg/ability/finance"
	provider "github.com/flowline-io/flowbot/pkg/providers/fireflyiii"
	"github.com/flowline-io/flowbot/pkg/types"
)

type client interface {
	CreateTransaction(transaction provider.Transaction) (*provider.TransactionResult, error)
}

type Adapter struct {
	client client
}

func New() finance.Service {
	return NewWithClient(provider.GetClient())
}

func NewWithClient(client client) finance.Service {
	return &Adapter{client: client}
}

func (a *Adapter) CreateTransaction(ctx context.Context, req finance.CreateTransactionRequest) (map[string]any, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "finance create transaction canceled", err)
	}
	if a.client == nil {
		return nil, types.Errorf(types.ErrUnavailable, "fireflyiii client not available")
	}
	srcID := 1
	if req.SourceID != "" {
		if n, err := strconv.Atoi(req.SourceID); err == nil {
			srcID = n
		}
	}
	transaction := provider.Transaction{
		ApplyRules:   true,
		FireWebhooks: true,
		Transactions: []provider.TransactionRecord{
			{
				Type:            string(provider.Withdrawal),
				Date:            req.Date,
				Amount:          req.Amount,
				Description:     req.Description,
				SourceId:        strconv.Itoa(srcID),
				DestinationName: "",
			},
		},
	}
	result, err := a.client.CreateTransaction(transaction)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "fireflyiii create transaction", err)
	}
	return map[string]any{
		"success": true,
		"result":  result,
	}, nil
}
