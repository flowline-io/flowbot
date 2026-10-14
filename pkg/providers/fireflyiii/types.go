package fireflyiii

import (
	"fmt"
	"time"

	"errors"
	"github.com/bytedance/sonic"
)

func ConvertResponseData[T any](resp *Response, statusCode int) (*T, error) {
	if resp == nil {
		return nil, errors.New("response is nil")
	}

	if statusCode != 200 {
		return nil, fmt.Errorf("response status code %d, message: %s, exception: %s", statusCode, resp.Message, resp.Exception)
	}

	if resp.Data == nil {
		return nil, fmt.Errorf("response data is empty (status %d, message: %s)", statusCode, resp.Message)
	}

	var result T
	b, err := sonic.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("marshal data failed: %w", err)
	}

	if err := sonic.Unmarshal(b, &result); err != nil {
		return nil, fmt.Errorf("unmarshal to target type failed: %w", err)
	}
	return &result, nil
}

// ConvertListResponseData unmarshals a Firefly list payload into typed items plus pagination.
func ConvertListResponseData[T any](resp *Response, statusCode int) (*ListResult[T], error) {
	if resp == nil {
		return nil, errors.New("response is nil")
	}
	if statusCode != 200 {
		return nil, fmt.Errorf("response status code %d, message: %s, exception: %s", statusCode, resp.Message, resp.Exception)
	}
	if resp.Data == nil {
		return nil, fmt.Errorf("response data is empty (status %d, message: %s)", statusCode, resp.Message)
	}

	var items []T
	b, err := sonic.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("marshal data failed: %w", err)
	}
	if err := sonic.Unmarshal(b, &items); err != nil {
		return nil, fmt.Errorf("unmarshal list data failed: %w", err)
	}
	out := &ListResult[T]{Items: items}
	if resp.Meta != nil {
		out.Pagination = resp.Meta.Pagination
	}
	return out, nil
}

type Response struct {
	Data      any        `json:"data"`
	Meta      *Meta      `json:"meta,omitempty"`
	Message   string     `json:"message"`
	Exception string     `json:"exception"`
}

// Meta holds Firefly III list response metadata.
type Meta struct {
	Pagination Pagination `json:"pagination"`
}

// Pagination holds Firefly III page-number pagination fields.
type Pagination struct {
	Total       int `json:"total"`
	Count       int `json:"count"`
	PerPage     int `json:"per_page"`
	CurrentPage int `json:"current_page"`
	TotalPages  int `json:"total_pages"`
}

// ListResult is a typed list page from Firefly III.
type ListResult[T any] struct {
	Items      []T
	Pagination Pagination
}

// ListQuery holds shared page-number list options.
type ListQuery struct {
	Page    int
	PerPage int
}

// TransactionListQuery holds filters for listing transactions.
type TransactionListQuery struct {
	ListQuery
	Type      string
	Start     string
	End       string
	AccountID string
}

// AccountListQuery holds filters for listing accounts.
type AccountListQuery struct {
	ListQuery
	Type string
}

const (
	// DefaultPerPage is the default page size for list requests.
	DefaultPerPage = 50
	// MaxPerPage is the maximum page size accepted by the provider client.
	MaxPerPage = 100
)

type About struct {
	Version    string `json:"version"`
	ApiVersion string `json:"api_version"`
	PhpVersion string `json:"php_version"`
	Os         string `json:"os"`
	Driver     string `json:"driver"`
}

type UserAttributes struct {
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Email       string    `json:"email"`
	Blocked     bool      `json:"blocked"`
	BlockedCode string    `json:"blocked_code"`
	Role        string    `json:"role"`
}

type User struct {
	Type       string         `json:"type"`
	Id         string         `json:"id"`
	Attributes UserAttributes `json:"attributes"`
}

type TransactionResultTransaction struct {
	User                         string    `json:"user"`
	TransactionJournalId         string    `json:"transaction_journal_id"`
	Type                         string    `json:"type"`
	Date                         time.Time `json:"date"`
	Order                        int       `json:"order"`
	CurrencyId                   string    `json:"currency_id"`
	CurrencyCode                 string    `json:"currency_code"`
	CurrencySymbol               string    `json:"currency_symbol"`
	CurrencyName                 string    `json:"currency_name"`
	CurrencyDecimalPlaces        int       `json:"currency_decimal_places"`
	ForeignCurrencyId            string    `json:"foreign_currency_id"`
	ForeignCurrencyCode          string    `json:"foreign_currency_code"`
	ForeignCurrencySymbol        string    `json:"foreign_currency_symbol"`
	ForeignCurrencyDecimalPlaces int       `json:"foreign_currency_decimal_places"`
	Amount                       string    `json:"amount"`
	ForeignAmount                string    `json:"foreign_amount"`
	Description                  string    `json:"description"`
	SourceId                     string    `json:"source_id"`
	SourceName                   string    `json:"source_name"`
	SourceIban                   string    `json:"source_iban"`
	SourceType                   string    `json:"source_type"`
	DestinationId                string    `json:"destination_id"`
	DestinationName              string    `json:"destination_name"`
	DestinationIban              string    `json:"destination_iban"`
	DestinationType              string    `json:"destination_type"`
	BudgetId                     string    `json:"budget_id"`
	BudgetName                   string    `json:"budget_name"`
	CategoryId                   string    `json:"category_id"`
	CategoryName                 string    `json:"category_name"`
	BillId                       string    `json:"bill_id"`
	BillName                     string    `json:"bill_name"`
	Reconciled                   bool      `json:"reconciled"`
	Notes                        string    `json:"notes"`
	Tags                         any       `json:"tags"`
	InternalReference            string    `json:"internal_reference"`
	ExternalId                   string    `json:"external_id"`
	ExternalUrl                  string    `json:"external_url"`
	OriginalSource               string    `json:"original_source"`
	RecurrenceId                 string    `json:"recurrence_id"`
	RecurrenceTotal              int       `json:"recurrence_total"`
	RecurrenceCount              int       `json:"recurrence_count"`
	BunqPaymentId                string    `json:"bunq_payment_id"`
	ImportHashV2                 string    `json:"import_hash_v2"`
	SepaCc                       string    `json:"sepa_cc"`
	SepaCtOp                     string    `json:"sepa_ct_op"`
	SepaCtId                     string    `json:"sepa_ct_id"`
	SepaDb                       string    `json:"sepa_db"`
	SepaCountry                  string    `json:"sepa_country"`
	SepaEp                       string    `json:"sepa_ep"`
	SepaCi                       string    `json:"sepa_ci"`
	SepaBatchId                  string    `json:"sepa_batch_id"`
	InterestDate                 time.Time `json:"interest_date"`
	BookDate                     time.Time `json:"book_date"`
	ProcessDate                  time.Time `json:"process_date"`
	DueDate                      time.Time `json:"due_date"`
	PaymentDate                  time.Time `json:"payment_date"`
	InvoiceDate                  time.Time `json:"invoice_date"`
	Latitude                     float64   `json:"latitude"`
	Longitude                    float64   `json:"longitude"`
	ZoomLevel                    int       `json:"zoom_level"`
	HasAttachments               bool      `json:"has_attachments"`
}

type TransactionResultAttributes struct {
	CreatedAt    time.Time                      `json:"created_at"`
	UpdatedAt    time.Time                      `json:"updated_at"`
	User         string                         `json:"user"`
	GroupTitle   string                         `json:"group_title"`
	Transactions []TransactionResultTransaction `json:"transactions"`
}

type TransactionResult struct {
	Type       string                      `json:"type"`
	Id         string                      `json:"id"`
	Attributes TransactionResultAttributes `json:"attributes"`
}

type Transaction struct {
	ApplyRules   bool                `json:"apply_rules"`
	FireWebhooks bool                `json:"fire_webhooks"`
	Transactions []TransactionRecord `json:"transactions"`
}

type TransactionRecord struct {
	TransactionJournalId string   `json:"transaction_journal_id,omitempty"`
	Type                 string   `json:"type"`
	Date                 string   `json:"date"`
	Amount               string   `json:"amount"`
	Description          string   `json:"description"`
	SourceId             string   `json:"source_id,omitempty"`
	SourceName           string   `json:"source_name,omitempty"`
	DestinationId        string   `json:"destination_id,omitempty"`
	DestinationName      string   `json:"destination_name,omitempty"`
	CategoryName         string   `json:"category_name,omitempty"`
	BudgetId             string   `json:"budget_id,omitempty"`
	BudgetName           string   `json:"budget_name,omitempty"`
	BillId               string    `json:"bill_id,omitempty"`
	BillName             string    `json:"bill_name,omitempty"`
	Tags                 *[]string `json:"tags,omitempty"`
	InterestDate         string    `json:"interest_date,omitempty"`
	BookDate             string    `json:"book_date,omitempty"`
	ProcessDate          string    `json:"process_date,omitempty"`
	DueDate              string    `json:"due_date,omitempty"`
	PaymentDate          string    `json:"payment_date,omitempty"`
	InvoiceDate          string    `json:"invoice_date,omitempty"`
	InternalReference    string    `json:"internal_reference,omitempty"`
	Notes                string    `json:"notes,omitempty"`
	ExternalUrl          string    `json:"external_url,omitempty"`
}

// Account is a Firefly III account resource.
type Account struct {
	Type       string           `json:"type"`
	Id         string           `json:"id"`
	Attributes AccountAttributes `json:"attributes"`
}

// AccountAttributes holds account fields used by the capability layer.
type AccountAttributes struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// Category is a Firefly III category resource.
type Category struct {
	Type       string             `json:"type"`
	Id         string             `json:"id"`
	Attributes CategoryAttributes `json:"attributes"`
}

// CategoryAttributes holds category fields used by the capability layer.
type CategoryAttributes struct {
	Name string `json:"name"`
}

// Budget is a Firefly III budget resource.
type Budget struct {
	Type       string           `json:"type"`
	Id         string           `json:"id"`
	Attributes BudgetAttributes `json:"attributes"`
}

// BudgetAttributes holds budget fields used by the capability layer.
type BudgetAttributes struct {
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

// Bill is a Firefly III bill resource.
type Bill struct {
	Type       string         `json:"type"`
	Id         string         `json:"id"`
	Attributes BillAttributes `json:"attributes"`
}

// BillAttributes holds bill fields used by the capability layer.
type BillAttributes struct {
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

// Tag is a Firefly III tag resource.
type Tag struct {
	Type       string        `json:"type"`
	Id         string        `json:"id"`
	Attributes TagAttributes `json:"attributes"`
}

// TagAttributes holds tag fields used by the capability layer.
type TagAttributes struct {
	Tag string `json:"tag"`
}

type TransactionType string

const (
	Withdrawal     TransactionType = "withdrawal"
	Deposit        TransactionType = "deposit"
	Transfer       TransactionType = "transfer"
	Reconciliation TransactionType = "reconciliation"
	OpeningBalance TransactionType = "opening balance"
)
