package nocodb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
)

type mockService struct{}

func (*mockService) ListBases(_ context.Context) (*capability.ListResult[capability.NocoBase], error) {
	return &capability.ListResult[capability.NocoBase]{
		Items: []*capability.NocoBase{{ID: "b1"}},
		Page:  &capability.PageInfo{Limit: 25},
	}, nil
}
func (*mockService) ListTables(_ context.Context, _ ListTablesInput) (*capability.ListResult[capability.NocoTable], error) {
	return &capability.ListResult[capability.NocoTable]{Items: []*capability.NocoTable{{ID: "t1"}}}, nil
}
func (*mockService) GetTable(_ context.Context, _ GetTableInput) (*capability.NocoTable, error) {
	return &capability.NocoTable{ID: "t1"}, nil
}
func (*mockService) ListRecords(_ context.Context, _ ListRecordsInput) (*capability.ListResult[capability.NocoRecord], error) {
	total := int64(1)
	return &capability.ListResult[capability.NocoRecord]{
		Items: []*capability.NocoRecord{{ID: "1"}},
		Page:  &capability.PageInfo{Limit: 10, HasMore: false, Total: &total},
	}, nil
}
func (*mockService) GetRecord(_ context.Context, _ GetRecordInput) (*capability.NocoRecord, error) {
	return &capability.NocoRecord{ID: "1"}, nil
}
func (*mockService) CreateRecord(_ context.Context, _ CreateRecordInput) (*capability.NocoRecord, error) {
	return &capability.NocoRecord{ID: "2"}, nil
}
func (*mockService) UpdateRecord(_ context.Context, _ UpdateRecordInput) (*capability.NocoRecord, error) {
	return &capability.NocoRecord{ID: "2"}, nil
}
func (*mockService) DeleteRecord(_ context.Context, _ DeleteRecordInput) error { return nil }
func (*mockService) HealthCheck(_ context.Context) (bool, error)               { return true, nil }

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
	require.NoError(t, Register("nocodb", &mockService{}))
	desc, ok := hub.Default.Get(hub.CapNocodb)
	require.True(t, ok)
	assert.Equal(t, hub.CapNocodb, desc.Type)
	assert.Len(t, desc.Operations, 9)

	tests := []struct {
		name string
		op   string
	}{
		{"has list_bases", OpListBases},
		{"has list_tables", OpListTables},
		{"has get_table", OpGetTable},
		{"has list_records", OpListRecords},
		{"has get_record", OpGetRecord},
		{"has create_record", OpCreateRecord},
		{"has update_record", OpUpdateRecord},
		{"has delete_record", OpDeleteRecord},
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

func TestInvokeHandlers(t *testing.T) {
	require.NoError(t, Register("nocodb", &mockService{}))
	tests := []struct {
		name    string
		op      string
		params  map[string]any
		wantErr bool
	}{
		{name: "list bases", op: OpListBases, params: map[string]any{}},
		{name: "list tables requires base_id", op: OpListTables, params: map[string]any{}, wantErr: true},
		{name: "list tables success", op: OpListTables, params: map[string]any{"base_id": "b1"}},
		{name: "list records with page", op: OpListRecords, params: map[string]any{"table_id": "t1"}},
		{name: "create record", op: OpCreateRecord, params: map[string]any{
			"table_id": "t1", "fields": map[string]any{"Name": "a"},
		}},
		{name: "health", op: OpHealth, params: map[string]any{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := capability.Invoke(context.Background(), hub.CapNocodb, tt.op, tt.params)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, res)
			if tt.op == OpListRecords {
				require.NotNil(t, res.Page)
			}
		})
	}
}
