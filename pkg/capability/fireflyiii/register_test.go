package fireflyiii

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
)

type mockService struct{}

func (*mockService) CreateTransaction(_ context.Context, _ CreateTransactionInput) (*capability.Transaction, error) {
	return &capability.Transaction{ID: "1"}, nil
}
func (*mockService) About(_ context.Context) (*capability.FinanceAbout, error) {
	return &capability.FinanceAbout{Version: "6.0.0"}, nil
}
func (*mockService) CurrentUser(_ context.Context) (*capability.FinanceUser, error) {
	return &capability.FinanceUser{ID: "1"}, nil
}
func (*mockService) HealthCheck(_ context.Context) (bool, error) { return true, nil }

func TestRegister(t *testing.T) {
	tests := []struct {
		name    string
		app     string
		svc     Service
		wantErr bool
	}{
		{name: "nil service skips registration", app: "app1", svc: nil, wantErr: false},
		{name: "valid service", app: "app1", svc: &mockService{}, wantErr: false},
		{name: "empty app with valid service", app: "", svc: &mockService{}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Register(tt.app, tt.svc)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRegister_Operations(t *testing.T) {
	require.NoError(t, Register("fireflyiii", &mockService{}))
	desc, ok := hub.Default.Get(hub.CapFireflyiii)
	require.True(t, ok)
	assert.Equal(t, hub.CapFireflyiii, desc.Type)
	assert.Equal(t, "fireflyiii", desc.App)
	assert.True(t, desc.Healthy)
	assert.Len(t, desc.Operations, 4)

	tests := []struct {
		name string
		op   string
	}{
		{"has create_transaction", OpCreateTransaction},
		{"has about", OpAbout},
		{"has current_user", OpCurrentUser},
		{"has health", OpHealth},
	}
	opNames := make([]string, len(desc.Operations))
	for i, op := range desc.Operations {
		opNames[i] = op.Name
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Contains(t, opNames, tt.op)
		})
	}
}

func TestInvokeCreateTransaction(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		params  map[string]any
		wantErr bool
	}{
		{
			name: "success",
			params: map[string]any{
				"type": "withdrawal", "date": "2026-07-18", "amount": "10", "description": "test",
				"source_name": "Cash", "destination_name": "Store",
			},
		},
		{
			name:    "missing type",
			params:  map[string]any{"date": "2026-07-18", "amount": "10", "description": "test"},
			wantErr: true,
		},
		{
			name:    "missing amount",
			params:  map[string]any{"type": "withdrawal", "date": "2026-07-18", "description": "test"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			inv := invokeCreateTransaction(&mockService{}, "fireflyiii")
			res, err := inv(context.Background(), tt.params)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, res)
			assert.NotNil(t, res.Data)
		})
	}
}
