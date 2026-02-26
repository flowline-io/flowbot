package fireflyiii

import (
	"fmt"
	"time"

	"github.com/bytedance/sonic"
)

func ConvertResponseData[T any](resp *Response, statusCode int) (*T, error) {
	if resp == nil {
		return nil, fmt.Errorf("response is nil")
	}

	if statusCode != 200 {
		return nil, fmt.Errorf("response status code %d, message: %s, exception: %s", statusCode, resp.Message, resp.Exception)
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

type Response struct {
	Data      any    `json:"data"`
	Message   string `json:"message"`
	Exception string `json:"exception"`
}

type About struct {
	Version    string `json:"version"`
	ApiVersion string `json:"api_version"`
	PhpVersion string `json:"php_version"`
	Os         string `json:"os"`
	Driver     string `json:"driver"`
}

type User struct {
	Type       string `json:"type"`
	Id         string `json:"id"`
	Attributes struct {
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
		Email       string    `json:"email"`
		Blocked     bool      `json:"blocked"`
		BlockedCode string    `json:"blocked_code"`
		Role        string    `json:"role"`
	} `json:"attributes"`
}

type TransactionResult struct {
	Type       string `json:"type"`
	Id         string `json:"id"`
	Attributes struct {
		CreatedAt    time.Time `json:"created_at"`
		UpdatedAt    time.Time `json:"updated_at"`
		User         string    `json:"user"`
		GroupTitle   string    `json:"group_title"`
		Transactions []struct {
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
		} `json:"transactions"`
	} `json:"attributes"`
}

type Transaction struct {
	ApplyRules   bool                `json:"apply_rules"`
	FireWebhooks bool                `json:"fire_webhooks"`
	Transactions []TransactionRecord `json:"transactions"`
}

type TransactionRecord struct {
	Type              string `json:"type"`
	Date              string `json:"date"`
	Amount            string `json:"amount"`
	Description       string `json:"description"`
	SourceId          string `json:"source_id"`
	SourceName        string `json:"source_name"`
	DestinationId     int    `json:"destination_id"`
	DestinationName   string `json:"destination_name"`
	CategoryName      string `json:"category_name"`
	InterestDate      string `json:"interest_date"`
	BookDate          string `json:"book_date"`
	ProcessDate       string `json:"process_date"`
	DueDate           string `json:"due_date"`
	PaymentDate       string `json:"payment_date"`
	InvoiceDate       string `json:"invoice_date"`
	InternalReference string `json:"internal_reference"`
	Notes             string `json:"notes"`
	ExternalUrl       string `json:"external_url"`
}

type TransactionType string

const (
	Withdrawal     TransactionType = "withdrawal"
	Deposit        TransactionType = "deposit"
	Transfer       TransactionType = "transfer"
	Reconciliation TransactionType = "reconciliation"
	OpeningBalance TransactionType = "opening balance"
)
