package fireflyiii

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

var _ client = (*fakeClient)(nil)

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
				createResp: &provider.TransactionResult{
					Id:   "123",
					Type: "transactions",
					Attributes: provider.TransactionResultAttributes{
						Transactions: []provider.TransactionResultTransaction{
							{
								Type:            "withdrawal",
								Amount:          "10.00",
								Description:     "Groceries",
								SourceName:      "Cash",
								DestinationName: "Store",
								CategoryName:    "Food",
								CurrencyCode:    "USD",
								Date:            time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC),
							},
						},
					},
				},
			},
			input: CreateTransactionInput{
				Type:            "withdrawal",
				Date:            "2026-07-18",
				Amount:          "10.00",
				Description:     "Groceries",
				SourceName:      "Cash",
				DestinationName: "Store",
				CategoryName:    "Food",
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
			assert.True(t, tt.client.createCalled)
			require.NotNil(t, tt.client.lastCreate)
			require.Len(t, tt.client.lastCreate.Transactions, 1)
			assert.Equal(t, tt.input.DestinationName, tt.client.lastCreate.Transactions[0].DestinationName)
			assert.Equal(t, tt.input.DestinationID, tt.client.lastCreate.Transactions[0].DestinationId)
			assert.Equal(t, tt.input.SourceName, tt.client.lastCreate.Transactions[0].SourceName)
		})
	}
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
	// Without provider config loaded, GetClient returns nil and New should too.
	// This is a smoke check that New does not panic.
	_ = New()
}
