// Package fireflyiii implements the Firefly III finance provider.
package fireflyiii

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/utils"

	"resty.dev/v3"

	"github.com/flowline-io/flowbot/pkg/providers"
)

const (
	ID          = "fireflyiii"
	EndpointKey = "endpoint"
	TokenKey    = "token"
)

type FireflyIII struct {
	c *resty.Client
}

func GetClient() *FireflyIII {
	endpoint, _ := providers.GetConfig(ID, EndpointKey)
	tokenKey, _ := providers.GetConfig(ID, TokenKey)
	if endpoint.String() == "" {
		return nil
	}
	return NewFireflyIII(endpoint.String(), tokenKey.String())
}

// NewFireflyIII creates a Firefly III client with the given endpoint and token.
// If endpoint is empty, it returns nil.
func NewFireflyIII(endpoint, token string) *FireflyIII {
	if endpoint == "" {
		return nil
	}
	v := &FireflyIII{}

	v.c = utils.DefaultRestyClient()
	v.c.SetBaseURL(endpoint)
	v.c.SetAuthToken(token)
	// Firefly redirects unauthenticated browser-style requests (no Accept) to /login HTML.
	// Prefer JSON so auth failures surface as 401 with message/exception instead of empty 200 bodies.
	v.c.SetHeader("Accept", "application/json, application/vnd.api+json")

	return v
}

func (i *FireflyIII) About() (*About, error) {
	resp, err := i.c.R().Get("/v1/about")
	if err != nil {
		return nil, fmt.Errorf("failed to get about: %w", err)
	}
	payload, err := decodeResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to get about: %w", err)
	}
	return ConvertResponseData[About](payload, resp.StatusCode())
}

// CurrentUser returns the currently authenticated user.
func (i *FireflyIII) CurrentUser() (*User, error) {
	resp, err := i.c.R().Get("/v1/about/user")
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}
	payload, err := decodeResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}
	return ConvertResponseData[User](payload, resp.StatusCode())
}

// CreateTransaction Creates a new transaction.
func (i *FireflyIII) CreateTransaction(transaction Transaction) (*TransactionResult, error) {
	resp, err := i.c.R().
		SetBody(transaction).
		Post("/v1/transactions")
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}
	payload, err := decodeResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}
	return ConvertResponseData[TransactionResult](payload, resp.StatusCode())
}

// ListTransactions lists transaction groups with optional filters.
func (i *FireflyIII) ListTransactions(q TransactionListQuery) (*ListResult[TransactionResult], error) {
	req := i.c.R()
	applyListQuery(req, q.ListQuery)
	if q.Type != "" {
		req.SetQueryParam("type", q.Type)
	}
	if q.Start != "" {
		req.SetQueryParam("start", q.Start)
	}
	if q.End != "" {
		req.SetQueryParam("end", q.End)
	}
	if q.AccountID != "" {
		req.SetQueryParam("account_id", q.AccountID)
	}
	resp, err := req.Get("/v1/transactions")
	if err != nil {
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}
	payload, err := decodeResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}
	return ConvertListResponseData[TransactionResult](payload, resp.StatusCode())
}

// GetTransaction returns one transaction group by id.
func (i *FireflyIII) GetTransaction(id string) (*TransactionResult, error) {
	if id == "" {
		return nil, errors.New("transaction id is required")
	}
	resp, err := i.c.R().Get("/v1/transactions/" + id)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}
	payload, err := decodeResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}
	return ConvertResponseData[TransactionResult](payload, resp.StatusCode())
}

// UpdateTransaction replaces a transaction group by id.
func (i *FireflyIII) UpdateTransaction(id string, transaction Transaction) (*TransactionResult, error) {
	if id == "" {
		return nil, errors.New("transaction id is required")
	}
	resp, err := i.c.R().
		SetBody(transaction).
		Put("/v1/transactions/" + id)
	if err != nil {
		return nil, fmt.Errorf("failed to update transaction: %w", err)
	}
	payload, err := decodeResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to update transaction: %w", err)
	}
	return ConvertResponseData[TransactionResult](payload, resp.StatusCode())
}

// DeleteTransaction deletes a transaction group by id.
func (i *FireflyIII) DeleteTransaction(id string) error {
	if id == "" {
		return errors.New("transaction id is required")
	}
	resp, err := i.c.R().Delete("/v1/transactions/" + id)
	if err != nil {
		return fmt.Errorf("failed to delete transaction: %w", err)
	}
	code := resp.StatusCode()
	if code == 200 || code == 204 {
		return nil
	}
	payload, decErr := decodeResponse(resp)
	if decErr != nil {
		return fmt.Errorf("failed to delete transaction: status %d: %w", code, decErr)
	}
	return fmt.Errorf("failed to delete transaction: status %d, message: %s, exception: %s", code, payload.Message, payload.Exception)
}

// ListAccounts lists accounts with optional type filter.
func (i *FireflyIII) ListAccounts(q AccountListQuery) (*ListResult[Account], error) {
	req := i.c.R()
	applyListQuery(req, q.ListQuery)
	if q.Type != "" {
		req.SetQueryParam("type", q.Type)
	}
	resp, err := req.Get("/v1/accounts")
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}
	payload, err := decodeResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}
	return ConvertListResponseData[Account](payload, resp.StatusCode())
}

// ListCategories lists categories.
func (i *FireflyIII) ListCategories(q ListQuery) (*ListResult[Category], error) {
	return listResource[Category](i, "/v1/categories", q, "categories")
}

// ListBudgets lists budgets.
func (i *FireflyIII) ListBudgets(q ListQuery) (*ListResult[Budget], error) {
	return listResource[Budget](i, "/v1/budgets", q, "budgets")
}

// ListBills lists bills.
func (i *FireflyIII) ListBills(q ListQuery) (*ListResult[Bill], error) {
	return listResource[Bill](i, "/v1/bills", q, "bills")
}

// ListTags lists tags.
func (i *FireflyIII) ListTags(q ListQuery) (*ListResult[Tag], error) {
	return listResource[Tag](i, "/v1/tags", q, "tags")
}

func listResource[T any](i *FireflyIII, path string, q ListQuery, label string) (*ListResult[T], error) {
	req := i.c.R()
	applyListQuery(req, q)
	resp, err := req.Get(path)
	if err != nil {
		return nil, fmt.Errorf("failed to list %s: %w", label, err)
	}
	payload, err := decodeResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to list %s: %w", label, err)
	}
	return ConvertListResponseData[T](payload, resp.StatusCode())
}

func applyListQuery(req *resty.Request, q ListQuery) {
	page := q.Page
	if page <= 0 {
		page = 1
	}
	perPage := q.PerPage
	if perPage <= 0 {
		perPage = DefaultPerPage
	}
	if perPage > MaxPerPage {
		perPage = MaxPerPage
	}
	req.SetQueryParam("page", strconv.Itoa(page))
	req.SetQueryParam("limit", strconv.Itoa(perPage))
}

func decodeResponse(resp *resty.Response) (*Response, error) {
	if resp == nil {
		return nil, errors.New("response is nil")
	}
	body := resp.Bytes()
	if len(body) == 0 {
		return &Response{}, nil
	}
	var payload Response
	if err := sonic.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &payload, nil
}
