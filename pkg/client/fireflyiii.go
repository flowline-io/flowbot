package client

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/capability"
)

// FireflyiiiClient provides access to the Firefly III finance API.
type FireflyiiiClient struct {
	c *Client
}

// CreateTransactionRequest is the request body for creating a transaction.
type CreateTransactionRequest struct {
	Type            string `json:"type"`
	Date            string `json:"date"`
	Amount          string `json:"amount"`
	Description     string `json:"description"`
	SourceID        string `json:"source_id,omitempty"`
	SourceName      string `json:"source_name,omitempty"`
	DestinationID   string `json:"destination_id,omitempty"`
	DestinationName string `json:"destination_name,omitempty"`
	CategoryName    string `json:"category_name,omitempty"`
	Notes           string `json:"notes,omitempty"`
}

// TransactionItemResult holds a single transaction extracted from InvokeResult.
type TransactionItemResult struct {
	Item capability.Transaction `json:"data"`
}

// FinanceAboutResult holds about info extracted from InvokeResult.
type FinanceAboutResult struct {
	Item capability.FinanceAbout `json:"data"`
}

// FinanceUserResult holds user info extracted from InvokeResult.
type FinanceUserResult struct {
	Item capability.FinanceUser `json:"data"`
}

// FinanceHealthResult holds the health check result extracted from InvokeResult.
type FinanceHealthResult struct {
	Healthy bool `json:"data"`
}

// CreateTransaction creates a new finance transaction.
func (f *FireflyiiiClient) CreateTransaction(ctx context.Context, req *CreateTransactionRequest) (*capability.Transaction, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	if req.Type == "" {
		return nil, fmt.Errorf("type is required")
	}
	if req.Date == "" {
		return nil, fmt.Errorf("date is required")
	}
	if req.Amount == "" {
		return nil, fmt.Errorf("amount is required")
	}
	if req.Description == "" {
		return nil, fmt.Errorf("description is required")
	}
	if req.SourceID == "" && req.SourceName == "" {
		return nil, fmt.Errorf("source_id or source_name is required")
	}
	if req.DestinationID == "" && req.DestinationName == "" {
		return nil, fmt.Errorf("destination_id or destination_name is required")
	}
	var result TransactionItemResult
	err := f.c.Post(ctx, "/service/fireflyiii/transactions", req, &result)
	if err != nil {
		return nil, err
	}
	return &result.Item, nil
}

// About returns Firefly III instance metadata.
func (f *FireflyiiiClient) About(ctx context.Context) (*capability.FinanceAbout, error) {
	var result FinanceAboutResult
	err := f.c.Get(ctx, "/service/fireflyiii/about", &result)
	if err != nil {
		return nil, err
	}
	return &result.Item, nil
}

// CurrentUser returns the authenticated Firefly III user.
func (f *FireflyiiiClient) CurrentUser(ctx context.Context) (*capability.FinanceUser, error) {
	var result FinanceUserResult
	err := f.c.Get(ctx, "/service/fireflyiii/user", &result)
	if err != nil {
		return nil, err
	}
	return &result.Item, nil
}

// Health checks whether the Firefly III backend is reachable.
func (f *FireflyiiiClient) Health(ctx context.Context) (bool, error) {
	var result FinanceHealthResult
	err := f.c.Get(ctx, "/service/fireflyiii/health", &result)
	if err != nil {
		return false, err
	}
	return result.Healthy, nil
}
