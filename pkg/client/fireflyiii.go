package client

import (
	"context"
	"net/url"
	"strconv"

	"errors"
	"github.com/flowline-io/flowbot/pkg/capability"
)

// FireflyiiiClient provides access to the Firefly III finance API.
type FireflyiiiClient struct {
	c *Client
}

// CreateTransactionRequest is the request body for creating a transaction.
type CreateTransactionRequest struct {
	Type            string   `json:"type"`
	Date            string   `json:"date"`
	Amount          string   `json:"amount"`
	Description     string   `json:"description"`
	SourceID        string   `json:"source_id,omitempty"`
	SourceName      string   `json:"source_name,omitempty"`
	DestinationID   string   `json:"destination_id,omitempty"`
	DestinationName string   `json:"destination_name,omitempty"`
	CategoryName    string   `json:"category_name,omitempty"`
	BudgetID        string   `json:"budget_id,omitempty"`
	BudgetName      string   `json:"budget_name,omitempty"`
	BillID          string   `json:"bill_id,omitempty"`
	BillName        string   `json:"bill_name,omitempty"`
	Notes           string   `json:"notes,omitempty"`
	Tags            []string `json:"tags,omitempty"`
}

// UpdateTransactionFields is a partial update payload. Only keys present in the
// map are applied; empty strings and empty tag slices clear those fields.
type UpdateTransactionFields map[string]any

// ListTransactionsQuery holds filters for listing transactions.
type ListTransactionsQuery struct {
	Limit     int
	Cursor    string
	Type      string
	Start     string
	End       string
	AccountID string
}

// ListAccountsQuery holds filters for listing accounts.
type ListAccountsQuery struct {
	Limit  int
	Cursor string
	Type   string
}

// ListResourcesQuery holds pagination for simple finance resource lists.
type ListResourcesQuery struct {
	Limit  int
	Cursor string
}

// TransactionItemResult holds a single transaction extracted from InvokeResult.
type TransactionItemResult struct {
	Item capability.Transaction `json:"data"`
}

// TransactionListResult holds a paginated transaction list from InvokeResult.
type TransactionListResult struct {
	Items []*capability.Transaction `json:"data"`
	Page  FinancePage               `json:"page"`
}

// FinancePage holds pagination metadata for finance lists.
type FinancePage struct {
	Limit      int    `json:"limit"`
	HasMore    bool   `json:"has_more"`
	NextCursor string `json:"next_cursor,omitzero"`
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

// AccountListResult holds a paginated account list from InvokeResult.
type AccountListResult struct {
	Items []*capability.FinanceAccount `json:"data"`
	Page  FinancePage                  `json:"page"`
}

// CategoryListResult holds a paginated category list from InvokeResult.
type CategoryListResult struct {
	Items []*capability.FinanceCategory `json:"data"`
	Page  FinancePage                   `json:"page"`
}

// BudgetListResult holds a paginated budget list from InvokeResult.
type BudgetListResult struct {
	Items []*capability.FinanceBudget `json:"data"`
	Page  FinancePage                 `json:"page"`
}

// BillListResult holds a paginated bill list from InvokeResult.
type BillListResult struct {
	Items []*capability.FinanceBill `json:"data"`
	Page  FinancePage               `json:"page"`
}

// TagListResult holds a paginated tag list from InvokeResult.
type TagListResult struct {
	Items []*capability.FinanceTag `json:"data"`
	Page  FinancePage              `json:"page"`
}

// CreateTransaction creates a new finance transaction.
func (f *FireflyiiiClient) CreateTransaction(ctx context.Context, req *CreateTransactionRequest) (*capability.Transaction, error) {
	if req == nil {
		return nil, errors.New("request is required")
	}
	if req.Type == "" {
		return nil, errors.New("type is required")
	}
	if req.Date == "" {
		return nil, errors.New("date is required")
	}
	if req.Amount == "" {
		return nil, errors.New("amount is required")
	}
	if req.Description == "" {
		return nil, errors.New("description is required")
	}
	if req.SourceID == "" && req.SourceName == "" {
		return nil, errors.New("source_id or source_name is required")
	}
	if req.DestinationID == "" && req.DestinationName == "" {
		return nil, errors.New("destination_id or destination_name is required")
	}
	var result TransactionItemResult
	err := f.c.Post(ctx, "/service/fireflyiii/transactions", req, &result)
	if err != nil {
		return nil, err
	}
	return &result.Item, nil
}

// ListTransactions lists transactions with optional filters.
func (f *FireflyiiiClient) ListTransactions(ctx context.Context, query *ListTransactionsQuery) (*TransactionListResult, error) {
	path := "/service/fireflyiii/transactions"
	if query != nil {
		v := url.Values{}
		if query.Limit > 0 {
			v.Set("limit", strconv.Itoa(query.Limit))
		}
		if query.Cursor != "" {
			v.Set("cursor", query.Cursor)
		}
		if query.Type != "" {
			v.Set("type", query.Type)
		}
		if query.Start != "" {
			v.Set("start", query.Start)
		}
		if query.End != "" {
			v.Set("end", query.End)
		}
		if query.AccountID != "" {
			v.Set("account_id", query.AccountID)
		}
		if len(v) > 0 {
			path = path + "?" + v.Encode()
		}
	}
	var result TransactionListResult
	err := f.c.Get(ctx, path, &result)
	return &result, err
}

// GetTransaction returns one transaction by group id.
func (f *FireflyiiiClient) GetTransaction(ctx context.Context, id string) (*capability.Transaction, error) {
	if id == "" {
		return nil, errors.New("id is required")
	}
	var result TransactionItemResult
	err := f.c.Get(ctx, "/service/fireflyiii/transactions/"+url.PathEscape(id), &result)
	if err != nil {
		return nil, err
	}
	return &result.Item, nil
}

// UpdateTransaction partially updates a transaction group.
func (f *FireflyiiiClient) UpdateTransaction(ctx context.Context, id string, fields UpdateTransactionFields) (*capability.Transaction, error) {
	if id == "" {
		return nil, errors.New("id is required")
	}
	if fields == nil {
		return nil, errors.New("fields are required")
	}
	var result TransactionItemResult
	err := f.c.Put(ctx, "/service/fireflyiii/transactions/"+url.PathEscape(id), fields, &result)
	if err != nil {
		return nil, err
	}
	return &result.Item, nil
}

// DeleteTransaction deletes a transaction group by id.
func (f *FireflyiiiClient) DeleteTransaction(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("id is required")
	}
	return f.c.Delete(ctx, "/service/fireflyiii/transactions/"+url.PathEscape(id), nil, nil)
}

// ListAccounts lists accounts with optional type filter.
func (f *FireflyiiiClient) ListAccounts(ctx context.Context, query *ListAccountsQuery) (*AccountListResult, error) {
	path := "/service/fireflyiii/accounts"
	if query != nil {
		v := url.Values{}
		if query.Limit > 0 {
			v.Set("limit", strconv.Itoa(query.Limit))
		}
		if query.Cursor != "" {
			v.Set("cursor", query.Cursor)
		}
		if query.Type != "" {
			v.Set("type", query.Type)
		}
		if len(v) > 0 {
			path = path + "?" + v.Encode()
		}
	}
	var result AccountListResult
	err := f.c.Get(ctx, path, &result)
	return &result, err
}

// ListCategories lists categories.
func (f *FireflyiiiClient) ListCategories(ctx context.Context, query *ListResourcesQuery) (*CategoryListResult, error) {
	var result CategoryListResult
	err := f.c.Get(ctx, resourceListPath("/service/fireflyiii/categories", query), &result)
	return &result, err
}

// ListBudgets lists budgets.
func (f *FireflyiiiClient) ListBudgets(ctx context.Context, query *ListResourcesQuery) (*BudgetListResult, error) {
	var result BudgetListResult
	err := f.c.Get(ctx, resourceListPath("/service/fireflyiii/budgets", query), &result)
	return &result, err
}

// ListBills lists bills.
func (f *FireflyiiiClient) ListBills(ctx context.Context, query *ListResourcesQuery) (*BillListResult, error) {
	var result BillListResult
	err := f.c.Get(ctx, resourceListPath("/service/fireflyiii/bills", query), &result)
	return &result, err
}

// ListTags lists tags.
func (f *FireflyiiiClient) ListTags(ctx context.Context, query *ListResourcesQuery) (*TagListResult, error) {
	var result TagListResult
	err := f.c.Get(ctx, resourceListPath("/service/fireflyiii/tags", query), &result)
	return &result, err
}

func resourceListPath(base string, query *ListResourcesQuery) string {
	if query == nil {
		return base
	}
	v := url.Values{}
	if query.Limit > 0 {
		v.Set("limit", strconv.Itoa(query.Limit))
	}
	if query.Cursor != "" {
		v.Set("cursor", query.Cursor)
	}
	if len(v) == 0 {
		return base
	}
	return base + "?" + v.Encode()
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
