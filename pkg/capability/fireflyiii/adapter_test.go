package fireflyiii

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	provider "github.com/flowline-io/flowbot/pkg/providers/fireflyiii"
)

type fakeClient struct {
	aboutResp    *provider.About
	aboutErr     error
	userResp     *provider.User
	userErr      error
	createResp   *provider.TransactionResult
	createErr    error
	lastCreate   *provider.Transaction
	createCalled bool

	listTxResp   *provider.ListResult[provider.TransactionResult]
	listTxByPage map[int]*provider.ListResult[provider.TransactionResult]
	listTxErr    error
	getTxResp    *provider.TransactionResult
	getTxErr     error
	updateResp   *provider.TransactionResult
	updateErr    error
	lastUpdate   *provider.Transaction
	deleteErr    error
	deletedID    string

	accountsResp   *provider.ListResult[provider.Account]
	accountsErr    error
	categoriesResp *provider.ListResult[provider.Category]
	categoriesErr  error
	budgetsResp    *provider.ListResult[provider.Budget]
	budgetsErr     error
	billsResp      *provider.ListResult[provider.Bill]
	billsErr       error
	tagsResp       *provider.ListResult[provider.Tag]
	tagsErr        error
}

func (f *fakeClient) About() (*provider.About, error) {
	return f.aboutResp, f.aboutErr
}

func (f *fakeClient) CurrentUser() (*provider.User, error) {
	return f.userResp, f.userErr
}

func (f *fakeClient) CreateTransaction(transaction provider.Transaction) (*provider.TransactionResult, error) {
	f.createCalled = true
	f.lastCreate = &transaction
	return f.createResp, f.createErr
}

func (f *fakeClient) ListTransactions(q provider.TransactionListQuery) (*provider.ListResult[provider.TransactionResult], error) {
	if f.listTxByPage != nil {
		if resp, ok := f.listTxByPage[q.Page]; ok {
			return resp, nil
		}
	}
	return f.listTxResp, f.listTxErr
}

func (f *fakeClient) GetTransaction(_ string) (*provider.TransactionResult, error) {
	return f.getTxResp, f.getTxErr
}

func (f *fakeClient) UpdateTransaction(_ string, transaction provider.Transaction) (*provider.TransactionResult, error) {
	f.lastUpdate = &transaction
	return f.updateResp, f.updateErr
}

func (f *fakeClient) DeleteTransaction(id string) error {
	f.deletedID = id
	return f.deleteErr
}

func (f *fakeClient) ListAccounts(_ provider.AccountListQuery) (*provider.ListResult[provider.Account], error) {
	return f.accountsResp, f.accountsErr
}

func (f *fakeClient) ListCategories(_ provider.ListQuery) (*provider.ListResult[provider.Category], error) {
	return f.categoriesResp, f.categoriesErr
}

func (f *fakeClient) ListBudgets(_ provider.ListQuery) (*provider.ListResult[provider.Budget], error) {
	return f.budgetsResp, f.budgetsErr
}

func (f *fakeClient) ListBills(_ provider.ListQuery) (*provider.ListResult[provider.Bill], error) {
	return f.billsResp, f.billsErr
}

func (f *fakeClient) ListTags(_ provider.ListQuery) (*provider.ListResult[provider.Tag], error) {
	return f.tagsResp, f.tagsErr
}

var _ client = (*fakeClient)(nil)

func sampleTxResult(id string) *provider.TransactionResult {
	return &provider.TransactionResult{
		Id:   id,
		Type: "transactions",
		Attributes: provider.TransactionResultAttributes{
			Transactions: []provider.TransactionResultTransaction{
				{
					TransactionJournalId: "j1",
					Type:                 "withdrawal",
					Amount:               "10.00",
					Description:          "Groceries",
					SourceId:             "1",
					SourceName:           "Cash",
					DestinationId:        "2",
					DestinationName:      "Store",
					CategoryName:         "Food",
					BudgetName:           "Monthly",
					BillName:             "",
					CurrencyCode:         "USD",
					Tags:                 []any{"home"},
					Date:                 time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC),
				},
			},
		},
	}
}

func TestAdapter_CreateTransaction(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		client     *fakeClient
		input      CreateTransactionInput
		wantID     string
		wantAmount string
		wantErr    bool
		errContain string
	}{
		{
			name: "create success",
			client: &fakeClient{
				createResp: sampleTxResult("123"),
			},
			input: CreateTransactionInput{
				Type:            "withdrawal",
				Date:            "2026-07-18",
				Amount:          "10.00",
				Description:     "Groceries",
				SourceName:      "Cash",
				DestinationName: "Store",
				CategoryName:    "Food",
				BudgetName:      "Monthly",
				Tags:            []string{"home"},
			},
			wantID:     "123",
			wantAmount: "10.00",
		},
		{
			name:   "missing type",
			client: &fakeClient{},
			input: CreateTransactionInput{
				Date: "2026-07-18", Amount: "1", Description: "x",
				SourceName: "Cash", DestinationName: "Store",
			},
			wantErr:    true,
			errContain: "type is required",
		},
		{
			name:   "missing amount",
			client: &fakeClient{},
			input: CreateTransactionInput{
				Type: "withdrawal", Date: "2026-07-18", Description: "x",
				SourceName: "Cash", DestinationName: "Store",
			},
			wantErr:    true,
			errContain: "amount is required",
		},
		{
			name:   "missing source",
			client: &fakeClient{},
			input: CreateTransactionInput{
				Type: "withdrawal", Date: "2026-07-18", Amount: "1", Description: "x",
				DestinationName: "Store",
			},
			wantErr:    true,
			errContain: "source_id or source_name is required",
		},
		{
			name:   "missing destination",
			client: &fakeClient{},
			input: CreateTransactionInput{
				Type: "withdrawal", Date: "2026-07-18", Amount: "1", Description: "x",
				SourceName: "Cash",
			},
			wantErr:    true,
			errContain: "destination_id or destination_name is required",
		},
		{
			name:   "provider error",
			client: &fakeClient{createErr: assert.AnError},
			input: CreateTransactionInput{
				Type: "withdrawal", Date: "2026-07-18", Amount: "1", Description: "x",
				SourceName: "Cash", DestinationName: "Store",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := NewWithClient(tt.client)
			got, err := svc.CreateTransaction(context.Background(), tt.input)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantID, got.ID)
			assert.Equal(t, tt.wantAmount, got.Amount)
			assert.Equal(t, []string{"home"}, got.Tags)
			assert.True(t, tt.client.createCalled)
			require.NotNil(t, tt.client.lastCreate)
			require.Len(t, tt.client.lastCreate.Transactions, 1)
			assert.Equal(t, tt.input.DestinationName, tt.client.lastCreate.Transactions[0].DestinationName)
			assert.Equal(t, tt.input.BudgetName, tt.client.lastCreate.Transactions[0].BudgetName)
		})
	}
}

func TestAdapter_ListGetUpdateDelete(t *testing.T) {
	t.Parallel()
	t.Run("list transactions with next page", func(t *testing.T) {
		t.Parallel()
		fc := &fakeClient{
			listTxResp: &provider.ListResult[provider.TransactionResult]{
				Items:      []provider.TransactionResult{*sampleTxResult("10")},
				Pagination: provider.Pagination{CurrentPage: 1, TotalPages: 2, PerPage: 50},
			},
		}
		adapter, ok := NewWithClient(fc).(*Adapter)
		require.True(t, ok)
		adapter.SetCursorSecret([]byte("test-secret-xxxxxxxxxxxxxxxx"))
		result, err := adapter.ListTransactions(context.Background(), TransactionListQuery{
			Page: capability.PageRequest{Limit: 50},
			Type: "withdrawal",
		})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		assert.True(t, result.Page.HasMore)
		assert.NotEmpty(t, result.Page.NextCursor)
	})

	t.Run("get update delete", func(t *testing.T) {
		t.Parallel()
		amount := "12.00"
		fc := &fakeClient{
			getTxResp:  sampleTxResult("42"),
			updateResp: sampleTxResult("42"),
		}
		svc := NewWithClient(fc)
		got, err := svc.GetTransaction(context.Background(), "42")
		require.NoError(t, err)
		assert.Equal(t, "42", got.ID)
		assert.Equal(t, "1", got.SourceID)

		updated, err := svc.UpdateTransaction(context.Background(), UpdateTransactionInput{
			ID:     "42",
			Amount: &amount,
		})
		require.NoError(t, err)
		assert.Equal(t, "42", updated.ID)
		require.NotNil(t, fc.lastUpdate)
		assert.Equal(t, "12.00", fc.lastUpdate.Transactions[0].Amount)
		assert.Equal(t, "Groceries", fc.lastUpdate.Transactions[0].Description)

		require.NoError(t, svc.DeleteTransaction(context.Background(), "42"))
		assert.Equal(t, "42", fc.deletedID)
	})

	t.Run("list resources", func(t *testing.T) {
		t.Parallel()
		fc := &fakeClient{
			accountsResp: &provider.ListResult[provider.Account]{
				Items: []provider.Account{{Id: "1", Attributes: provider.AccountAttributes{Name: "Cash", Type: "asset"}}},
			},
			categoriesResp: &provider.ListResult[provider.Category]{
				Items: []provider.Category{{Id: "2", Attributes: provider.CategoryAttributes{Name: "Food"}}},
			},
			budgetsResp: &provider.ListResult[provider.Budget]{
				Items: []provider.Budget{{Id: "3", Attributes: provider.BudgetAttributes{Name: "Groceries", Active: true}}},
			},
			billsResp: &provider.ListResult[provider.Bill]{
				Items: []provider.Bill{{Id: "4", Attributes: provider.BillAttributes{Name: "Rent", Active: true}}},
			},
			tagsResp: &provider.ListResult[provider.Tag]{
				Items: []provider.Tag{{Id: "5", Attributes: provider.TagAttributes{Tag: "travel"}}},
			},
		}
		svc := NewWithClient(fc)
		accounts, err := svc.ListAccounts(context.Background(), AccountListQuery{Type: "asset"})
		require.NoError(t, err)
		assert.Equal(t, "Cash", accounts.Items[0].Name)

		categories, err := svc.ListCategories(context.Background(), ResourceListQuery{})
		require.NoError(t, err)
		assert.Equal(t, "Food", categories.Items[0].Name)

		budgets, err := svc.ListBudgets(context.Background(), ResourceListQuery{})
		require.NoError(t, err)
		assert.True(t, budgets.Items[0].Active)

		bills, err := svc.ListBills(context.Background(), ResourceListQuery{})
		require.NoError(t, err)
		assert.Equal(t, "Rent", bills.Items[0].Name)

		tags, err := svc.ListTags(context.Background(), ResourceListQuery{})
		require.NoError(t, err)
		assert.Equal(t, "travel", tags.Items[0].Name)
	})

	t.Run("update preserves non-first splits and clears tags", func(t *testing.T) {
		t.Parallel()
		existing := sampleTxResult("42")
		existing.Attributes.Transactions = append(existing.Attributes.Transactions, provider.TransactionResultTransaction{
			TransactionJournalId: "j2",
			Type:                 "withdrawal",
			Amount:               "5.00",
			Description:          "Split two",
			SourceId:             "1",
			SourceName:           "Cash",
			DestinationId:        "3",
			DestinationName:      "Other",
			Date:                 time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC),
		})
		emptyTags := []string{}
		notes := ""
		fc := &fakeClient{
			getTxResp:  existing,
			updateResp: sampleTxResult("42"),
		}
		svc := NewWithClient(fc)
		_, err := svc.UpdateTransaction(context.Background(), UpdateTransactionInput{
			ID:    "42",
			Notes: &notes,
			Tags:  &emptyTags,
		})
		require.NoError(t, err)
		require.NotNil(t, fc.lastUpdate)
		require.Len(t, fc.lastUpdate.Transactions, 2)
		assert.Equal(t, "j1", fc.lastUpdate.Transactions[0].TransactionJournalId)
		assert.Equal(t, "j2", fc.lastUpdate.Transactions[1].TransactionJournalId)
		assert.Equal(t, "Split two", fc.lastUpdate.Transactions[1].Description)
		assert.Empty(t, fc.lastUpdate.Transactions[0].Notes)
		require.NotNil(t, fc.lastUpdate.Transactions[0].Tags)
		assert.Empty(t, *fc.lastUpdate.Transactions[0].Tags)
	})
}

func TestAdapter_AboutUserHealth(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "about success",
			run: func(t *testing.T) {
				svc := NewWithClient(&fakeClient{aboutResp: &provider.About{Version: "6.0.0", Os: "Linux"}})
				info, err := svc.About(context.Background())
				require.NoError(t, err)
				assert.Equal(t, "6.0.0", info.Version)
				assert.Equal(t, "Linux", info.OS)
			},
		},
		{
			name: "about error",
			run: func(t *testing.T) {
				svc := NewWithClient(&fakeClient{aboutErr: assert.AnError})
				_, err := svc.About(context.Background())
				require.Error(t, err)
			},
		},
		{
			name: "current user success",
			run: func(t *testing.T) {
				svc := NewWithClient(&fakeClient{userResp: &provider.User{
					Id:         "1",
					Attributes: provider.UserAttributes{Email: "a@b.c", Role: "owner"},
				}})
				user, err := svc.CurrentUser(context.Background())
				require.NoError(t, err)
				assert.Equal(t, "1", user.ID)
				assert.Equal(t, "a@b.c", user.Email)
			},
		},
		{
			name: "health success",
			run: func(t *testing.T) {
				svc := NewWithClient(&fakeClient{aboutResp: &provider.About{Version: "6.0.0"}})
				ok, err := svc.HealthCheck(context.Background())
				require.NoError(t, err)
				assert.True(t, ok)
			},
		},
		{
			name: "health unhealthy",
			run: func(t *testing.T) {
				svc := NewWithClient(&fakeClient{aboutErr: assert.AnError})
				ok, err := svc.HealthCheck(context.Background())
				require.Error(t, err)
				assert.False(t, ok)
			},
		},
		{
			name: "toTransaction ignores resource type when nested empty",
			run: func(t *testing.T) {
				tx := toTransaction(&provider.TransactionResult{
					Id:   "99",
					Type: "transactions",
				})
				require.NotNil(t, tx)
				assert.Equal(t, "99", tx.ID)
				assert.Empty(t, tx.Type)
			},
		},
		{
			name: "canceled context",
			run: func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				svc := NewWithClient(&fakeClient{})
				_, err := svc.About(ctx)
				require.Error(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t)
		})
	}
}

func TestNew_NilWhenUnconfigured(t *testing.T) {
	t.Parallel()
	_ = New()
}

func TestAdapter_ListTransactionsTwoPagesDoNotRepeatIDs(t *testing.T) {
	t.Parallel()
	fc := &fakeClient{listTxByPage: map[int]*provider.ListResult[provider.TransactionResult]{
		1: {
			Items:      []provider.TransactionResult{*sampleTxResult("10")},
			Pagination: provider.Pagination{CurrentPage: 1, TotalPages: 2, PerPage: 1},
		},
		2: {
			Items:      []provider.TransactionResult{*sampleTxResult("11")},
			Pagination: provider.Pagination{CurrentPage: 2, TotalPages: 2, PerPage: 1},
		},
	}}
	adapter, ok := NewWithClient(fc).(*Adapter)
	require.True(t, ok)
	adapter.SetCursorSecret([]byte("test-secret-xxxxxxxxxxxxxxxx"))
	first, err := adapter.ListTransactions(context.Background(), TransactionListQuery{Page: capability.PageRequest{Limit: 1}})
	require.NoError(t, err)
	require.Len(t, first.Items, 1)
	require.NotEmpty(t, first.Page.NextCursor)
	second, err := adapter.ListTransactions(context.Background(), TransactionListQuery{
		Page: capability.PageRequest{Limit: 1, Cursor: first.Page.NextCursor},
	})
	require.NoError(t, err)
	require.Len(t, second.Items, 1)
	assert.NotEqual(t, first.Items[0].ID, second.Items[0].ID)
}
