// Package fireflyiii implements the Firefly III adapter for the finance capability.
package fireflyiii

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/capability"
	provider "github.com/flowline-io/flowbot/pkg/providers/fireflyiii"
	"github.com/flowline-io/flowbot/pkg/types"
)

// client defines the subset of provider.FireflyIII methods used by this adapter.
type client interface {
	About() (*provider.About, error)
	CurrentUser() (*provider.User, error)
	CreateTransaction(transaction provider.Transaction) (*provider.TransactionResult, error)
}

// Adapter implements Service using the Firefly III provider client.
type Adapter struct {
	client client
}

// New creates an Adapter using the default provider client (reads config from YAML).
// Returns nil when the provider is not configured.
func New() Service {
	if c := provider.GetClient(); c != nil {
		return NewWithClient(c)
	}
	return nil
}

// NewWithClient creates an Adapter with a specific client, useful for testing.
func NewWithClient(c client) Service {
	return &Adapter{client: c}
}

// CreateTransaction creates a new finance transaction.
// Source and destination must each be identified by id and/or name.
// ApplyRules and FireWebhooks are enabled so Firefly processes rules and notifies listeners.
func (a *Adapter) CreateTransaction(ctx context.Context, in CreateTransactionInput) (*capability.Transaction, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if in.Type == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "type is required")
	}
	if in.Amount == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "amount is required")
	}
	if in.Description == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "description is required")
	}
	if in.Date == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "date is required")
	}
	if in.SourceID == "" && in.SourceName == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "source_id or source_name is required")
	}
	if in.DestinationID == "" && in.DestinationName == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "destination_id or destination_name is required")
	}

	record := provider.TransactionRecord{
		Type:            in.Type,
		Date:            in.Date,
		Amount:          in.Amount,
		Description:     in.Description,
		SourceId:        in.SourceID,
		SourceName:      in.SourceName,
		DestinationId:   in.DestinationID,
		DestinationName: in.DestinationName,
		CategoryName:    in.CategoryName,
		Notes:           in.Notes,
	}

	result, err := a.client.CreateTransaction(provider.Transaction{
		ApplyRules:   true,
		FireWebhooks: true,
		Transactions: []provider.TransactionRecord{record},
	})
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "fireflyiii create transaction failed", err)
	}
	return toTransaction(result), nil
}

// About returns Firefly III instance metadata.
func (a *Adapter) About(ctx context.Context) (*capability.FinanceAbout, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	info, err := a.client.About()
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "fireflyiii about failed", err)
	}
	return toAbout(info), nil
}

// CurrentUser returns the authenticated Firefly III user.
func (a *Adapter) CurrentUser(ctx context.Context) (*capability.FinanceUser, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	user, err := a.client.CurrentUser()
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "fireflyiii current user failed", err)
	}
	return toUser(user), nil
}

// HealthCheck reports whether the Firefly III backend is reachable.
func (a *Adapter) HealthCheck(ctx context.Context) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	_, err := a.client.About()
	if err != nil {
		return false, types.WrapError(types.ErrProvider, "fireflyiii health check failed", err)
	}
	return true, nil
}

func toTransaction(r *provider.TransactionResult) *capability.Transaction {
	if r == nil {
		return nil
	}
	tx := &capability.Transaction{ID: r.Id}
	if len(r.Attributes.Transactions) == 0 {
		return tx
	}
	first := r.Attributes.Transactions[0]
	tx.Type = first.Type
	tx.Amount = first.Amount
	tx.Description = first.Description
	tx.CurrencyCode = first.CurrencyCode
	tx.SourceName = first.SourceName
	tx.DestinationName = first.DestinationName
	tx.CategoryName = first.CategoryName
	tx.Notes = first.Notes
	if !first.Date.IsZero() {
		tx.Date = first.Date.Format("2006-01-02")
	}
	return tx
}

func toAbout(a *provider.About) *capability.FinanceAbout {
	if a == nil {
		return nil
	}
	return &capability.FinanceAbout{
		Version:    a.Version,
		APIVersion: a.ApiVersion,
		PHPVersion: a.PhpVersion,
		OS:         a.Os,
		Driver:     a.Driver,
	}
}

func toUser(u *provider.User) *capability.FinanceUser {
	if u == nil {
		return nil
	}
	return &capability.FinanceUser{
		ID:    u.Id,
		Email: u.Attributes.Email,
		Role:  u.Attributes.Role,
	}
}

// Compile-time interface check.
var _ Service = (*Adapter)(nil)
