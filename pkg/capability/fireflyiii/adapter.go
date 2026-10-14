// Package fireflyiii implements the Firefly III adapter for the finance capability.
package fireflyiii

import (
	"context"
	"strconv"
	"time"

	"github.com/flowline-io/flowbot/pkg/capability"
	provider "github.com/flowline-io/flowbot/pkg/providers/fireflyiii"
	"github.com/flowline-io/flowbot/pkg/types"
)

var defaultCursorSecret = []byte("flowbot-ability-finance-fireflyiii-cursor-v1")

// client defines the subset of provider.FireflyIII methods used by this adapter.
type client interface {
	About() (*provider.About, error)
	CurrentUser() (*provider.User, error)
	CreateTransaction(transaction provider.Transaction) (*provider.TransactionResult, error)
	ListTransactions(q provider.TransactionListQuery) (*provider.ListResult[provider.TransactionResult], error)
	GetTransaction(id string) (*provider.TransactionResult, error)
	UpdateTransaction(id string, transaction provider.Transaction) (*provider.TransactionResult, error)
	DeleteTransaction(id string) error
	ListAccounts(q provider.AccountListQuery) (*provider.ListResult[provider.Account], error)
	ListCategories(q provider.ListQuery) (*provider.ListResult[provider.Category], error)
	ListBudgets(q provider.ListQuery) (*provider.ListResult[provider.Budget], error)
	ListBills(q provider.ListQuery) (*provider.ListResult[provider.Bill], error)
	ListTags(q provider.ListQuery) (*provider.ListResult[provider.Tag], error)
}

// Adapter implements Service using the Firefly III provider client.
type Adapter struct {
	client       client
	cursorSecret []byte
	now          func() time.Time
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
	return &Adapter{
		client:       c,
		cursorSecret: defaultCursorSecret,
		now:          time.Now,
	}
}

// SetCursorSecret overrides the HMAC secret used for opaque list cursors.
func (a *Adapter) SetCursorSecret(secret []byte) {
	a.cursorSecret = secret
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
		BudgetId:        in.BudgetID,
		BudgetName:      in.BudgetName,
		BillId:          in.BillID,
		BillName:        in.BillName,
		Notes:           in.Notes,
		Tags:            tagsPointer(in.Tags),
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

// ListTransactions lists transaction groups with filters and opaque cursor pagination.
func (a *Adapter) ListTransactions(ctx context.Context, q TransactionListQuery) (*capability.ListResult[capability.Transaction], error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	page, limit, err := a.decodePage(q.Page)
	if err != nil {
		return nil, err
	}
	resp, err := a.client.ListTransactions(provider.TransactionListQuery{
		ListQuery: provider.ListQuery{Page: page, PerPage: limit},
		Type:      q.Type,
		Start:     q.Start,
		End:       q.End,
		AccountID: q.AccountID,
	})
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "fireflyiii list transactions failed", err)
	}
	items := make([]*capability.Transaction, 0, len(resp.Items))
	for i := range resp.Items {
		items = append(items, toTransaction(&resp.Items[i]))
	}
	return a.listResult(items, limit, resp.Pagination)
}

// GetTransaction returns one transaction group by id.
func (a *Adapter) GetTransaction(ctx context.Context, id string) (*capability.Transaction, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if id == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	result, err := a.client.GetTransaction(id)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "fireflyiii get transaction failed", err)
	}
	return toTransaction(result), nil
}

// UpdateTransaction applies a partial update to the first split of a transaction group.
// All existing splits are sent back on PUT so Firefly does not drop non-first splits.
func (a *Adapter) UpdateTransaction(ctx context.Context, in UpdateTransactionInput) (*capability.Transaction, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if in.ID == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	existing, err := a.client.GetTransaction(in.ID)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "fireflyiii get transaction for update failed", err)
	}
	if existing == nil || len(existing.Attributes.Transactions) == 0 {
		return nil, types.Errorf(types.ErrNotFound, "transaction %s has no splits", in.ID)
	}
	records := make([]provider.TransactionRecord, 0, len(existing.Attributes.Transactions))
	for i := range existing.Attributes.Transactions {
		records = append(records, resultSplitToRecord(existing.Attributes.Transactions[i]))
	}
	applyUpdate(&records[0], in)
	result, err := a.client.UpdateTransaction(in.ID, provider.Transaction{
		ApplyRules:   true,
		FireWebhooks: true,
		Transactions: records,
	})
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "fireflyiii update transaction failed", err)
	}
	return toTransaction(result), nil
}

// DeleteTransaction deletes a transaction group by id.
func (a *Adapter) DeleteTransaction(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if id == "" {
		return types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	if err := a.client.DeleteTransaction(id); err != nil {
		return types.WrapError(types.ErrProvider, "fireflyiii delete transaction failed", err)
	}
	return nil
}

// ListAccounts lists accounts with optional type filter.
func (a *Adapter) ListAccounts(ctx context.Context, q AccountListQuery) (*capability.ListResult[capability.FinanceAccount], error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	page, limit, err := a.decodePage(q.Page)
	if err != nil {
		return nil, err
	}
	resp, err := a.client.ListAccounts(provider.AccountListQuery{
		ListQuery: provider.ListQuery{Page: page, PerPage: limit},
		Type:      q.Type,
	})
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "fireflyiii list accounts failed", err)
	}
	items := make([]*capability.FinanceAccount, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, &capability.FinanceAccount{
			ID:   item.Id,
			Name: item.Attributes.Name,
			Type: item.Attributes.Type,
		})
	}
	return a.listResult(items, limit, resp.Pagination)
}

// ListCategories lists categories.
func (a *Adapter) ListCategories(ctx context.Context, q ResourceListQuery) (*capability.ListResult[capability.FinanceCategory], error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	page, limit, err := a.decodePage(q.Page)
	if err != nil {
		return nil, err
	}
	resp, err := a.client.ListCategories(provider.ListQuery{Page: page, PerPage: limit})
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "fireflyiii list categories failed", err)
	}
	items := make([]*capability.FinanceCategory, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, &capability.FinanceCategory{ID: item.Id, Name: item.Attributes.Name})
	}
	return a.listResult(items, limit, resp.Pagination)
}

// ListBudgets lists budgets.
func (a *Adapter) ListBudgets(ctx context.Context, q ResourceListQuery) (*capability.ListResult[capability.FinanceBudget], error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	page, limit, err := a.decodePage(q.Page)
	if err != nil {
		return nil, err
	}
	resp, err := a.client.ListBudgets(provider.ListQuery{Page: page, PerPage: limit})
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "fireflyiii list budgets failed", err)
	}
	items := make([]*capability.FinanceBudget, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, &capability.FinanceBudget{
			ID: item.Id, Name: item.Attributes.Name, Active: item.Attributes.Active,
		})
	}
	return a.listResult(items, limit, resp.Pagination)
}

// ListBills lists bills.
func (a *Adapter) ListBills(ctx context.Context, q ResourceListQuery) (*capability.ListResult[capability.FinanceBill], error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	page, limit, err := a.decodePage(q.Page)
	if err != nil {
		return nil, err
	}
	resp, err := a.client.ListBills(provider.ListQuery{Page: page, PerPage: limit})
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "fireflyiii list bills failed", err)
	}
	items := make([]*capability.FinanceBill, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, &capability.FinanceBill{
			ID: item.Id, Name: item.Attributes.Name, Active: item.Attributes.Active,
		})
	}
	return a.listResult(items, limit, resp.Pagination)
}

// ListTags lists tags.
func (a *Adapter) ListTags(ctx context.Context, q ResourceListQuery) (*capability.ListResult[capability.FinanceTag], error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	page, limit, err := a.decodePage(q.Page)
	if err != nil {
		return nil, err
	}
	resp, err := a.client.ListTags(provider.ListQuery{Page: page, PerPage: limit})
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "fireflyiii list tags failed", err)
	}
	items := make([]*capability.FinanceTag, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, &capability.FinanceTag{ID: item.Id, Name: item.Attributes.Tag})
	}
	return a.listResult(items, limit, resp.Pagination)
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

func (a *Adapter) decodePage(page capability.PageRequest) (pageNum, limit int, err error) {
	limit = normalizedLimit(page.Limit)
	pageNum = 1
	if page.Cursor == "" {
		return pageNum, limit, nil
	}
	payload, err := capability.DecodeCursor(a.cursorSecret, page.Cursor, a.now())
	if err != nil {
		return 0, 0, err
	}
	if payload.Limit > 0 {
		limit = normalizedLimit(payload.Limit)
	}
	if payload.Page > 0 {
		pageNum = payload.Page
	} else if payload.ProviderCursor != "" {
		n, convErr := strconv.Atoi(payload.ProviderCursor)
		if convErr != nil || n <= 0 {
			return 0, 0, types.Errorf(types.ErrInvalidArgument, "invalid pagination cursor")
		}
		pageNum = n
	}
	return pageNum, limit, nil
}

func (a *Adapter) listResult[T any](items []*T, limit int, pag provider.Pagination) (*capability.ListResult[T], error) {
	page := &capability.PageInfo{
		Limit:   limit,
		HasMore: pag.CurrentPage > 0 && pag.CurrentPage < pag.TotalPages,
	}
	if page.HasMore {
		nextPage := pag.CurrentPage + 1
		cursor, err := capability.EncodeCursor(a.cursorSecret, capability.CursorPayload{
			Capability: "fireflyiii",
			Strategy:   "page",
			Page:       nextPage,
			Limit:      limit,
		})
		if err != nil {
			return nil, err
		}
		page.NextCursor = cursor
	}
	return &capability.ListResult[T]{Items: items, Page: page}, nil
}

func normalizedLimit(limit int) int {
	if limit <= 0 {
		return provider.DefaultPerPage
	}
	if limit > provider.MaxPerPage {
		return provider.MaxPerPage
	}
	return limit
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
	tx.SourceID = first.SourceId
	tx.SourceName = first.SourceName
	tx.DestinationID = first.DestinationId
	tx.DestinationName = first.DestinationName
	tx.CategoryName = first.CategoryName
	tx.BudgetName = first.BudgetName
	tx.BillName = first.BillName
	tx.Notes = first.Notes
	tx.Tags = tagsToStrings(first.Tags)
	if !first.Date.IsZero() {
		tx.Date = first.Date.Format("2006-01-02")
	}
	return tx
}

func resultSplitToRecord(first provider.TransactionResultTransaction) provider.TransactionRecord {
	tags := tagsToStrings(first.Tags)
	if tags == nil {
		tags = []string{}
	}
	record := provider.TransactionRecord{
		TransactionJournalId: first.TransactionJournalId,
		Type:                 first.Type,
		Amount:               first.Amount,
		Description:          first.Description,
		SourceId:             first.SourceId,
		SourceName:           first.SourceName,
		DestinationId:        first.DestinationId,
		DestinationName:      first.DestinationName,
		CategoryName:         first.CategoryName,
		BudgetId:             first.BudgetId,
		BudgetName:           first.BudgetName,
		BillId:               first.BillId,
		BillName:             first.BillName,
		Notes:                first.Notes,
		Tags:                 &tags,
	}
	if !first.Date.IsZero() {
		record.Date = first.Date.Format("2006-01-02")
	}
	return record
}

func applyUpdate(record *provider.TransactionRecord, in UpdateTransactionInput) {
	setStr := func(dst *string, src *string) {
		if src != nil {
			*dst = *src
		}
	}
	setStr(&record.Type, in.Type)
	setStr(&record.Date, in.Date)
	setStr(&record.Amount, in.Amount)
	setStr(&record.Description, in.Description)
	setStr(&record.SourceId, in.SourceID)
	setStr(&record.SourceName, in.SourceName)
	setStr(&record.DestinationId, in.DestinationID)
	setStr(&record.DestinationName, in.DestinationName)
	setStr(&record.CategoryName, in.CategoryName)
	setStr(&record.BudgetId, in.BudgetID)
	setStr(&record.BudgetName, in.BudgetName)
	setStr(&record.BillId, in.BillID)
	setStr(&record.BillName, in.BillName)
	setStr(&record.Notes, in.Notes)
	if in.Tags != nil {
		tags := append([]string(nil), (*in.Tags)...)
		record.Tags = &tags
	}
}

func tagsPointer(tags []string) *[]string {
	if len(tags) == 0 {
		return nil
	}
	out := append([]string(nil), tags...)
	return &out
}

func tagsToStrings(v any) []string {
	switch t := v.(type) {
	case nil:
		return nil
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
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
