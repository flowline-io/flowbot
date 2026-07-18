package nocodb

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	provider "github.com/flowline-io/flowbot/pkg/providers/nocodb"
	"github.com/flowline-io/flowbot/pkg/types"
)

type fakeClient struct {
	bases      *provider.BaseList
	basesErr   error
	tables     *provider.TableList
	tablesErr  error
	table      *provider.Table
	tableErr   error
	records    *provider.RecordList
	recordsErr error
	record     provider.Record
	recordErr  error
	createResp provider.Record
	createErr  error
	updateResp provider.Record
	updateErr  error
	updateBody map[string]any
	deleteErr  error
	deleteID   string
}

func (f *fakeClient) ListBases(_ context.Context) (*provider.BaseList, error) {
	return f.bases, f.basesErr
}
func (f *fakeClient) ListTables(_ context.Context, _ string) (*provider.TableList, error) {
	return f.tables, f.tablesErr
}
func (f *fakeClient) GetTable(_ context.Context, _ string) (*provider.Table, error) {
	return f.table, f.tableErr
}
func (f *fakeClient) ListRecords(_ context.Context, _ string, _ provider.ListRecordsQuery) (*provider.RecordList, error) {
	return f.records, f.recordsErr
}
func (f *fakeClient) GetRecord(_ context.Context, _, _ string) (provider.Record, error) {
	return f.record, f.recordErr
}
func (f *fakeClient) CreateRecord(_ context.Context, _ string, _ map[string]any) (provider.Record, error) {
	return f.createResp, f.createErr
}
func (f *fakeClient) UpdateRecord(_ context.Context, _ string, fields map[string]any) (provider.Record, error) {
	f.updateBody = fields
	return f.updateResp, f.updateErr
}
func (f *fakeClient) DeleteRecord(_ context.Context, _, id string) error {
	f.deleteID = id
	return f.deleteErr
}

var _ client = (*fakeClient)(nil)

func TestAdapter_ListBases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		client  *fakeClient
		wantLen int
		wantErr bool
	}{
		{
			name: "list success",
			client: &fakeClient{bases: &provider.BaseList{
				List:     []provider.Base{{ID: "b1", Title: "Home"}},
				PageInfo: provider.PageInfo{TotalRows: 1, PageSize: 25, IsLastPage: true},
			}},
			wantLen: 1,
		},
		{
			name:    "provider error",
			client:  &fakeClient{basesErr: assert.AnError},
			wantErr: true,
		},
		{
			name:    "nil result",
			client:  &fakeClient{bases: nil},
			wantLen: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := NewWithClient(tt.client)
			got, err := svc.ListBases(context.Background())
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Len(t, got.Items, tt.wantLen)
			require.NotNil(t, got.Page)
		})
	}
}

func TestAdapter_ListTables(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		client     *fakeClient
		input      ListTablesInput
		wantErr    bool
		errContain string
	}{
		{
			name:   "list success",
			client: &fakeClient{tables: &provider.TableList{List: []provider.Table{{ID: "t1", Title: "Tasks"}}}},
			input:  ListTablesInput{BaseID: "b1"},
		},
		{
			name:       "missing base id",
			client:     &fakeClient{},
			input:      ListTablesInput{},
			wantErr:    true,
			errContain: "base_id is required",
		},
		{
			name:    "provider error",
			client:  &fakeClient{tablesErr: assert.AnError},
			input:   ListTablesInput{BaseID: "b1"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := NewWithClient(tt.client)
			got, err := svc.ListTables(context.Background(), tt.input)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.Len(t, got.Items, 1)
			assert.Equal(t, "t1", got.Items[0].ID)
		})
	}
}

func TestAdapter_RecordsCRUD(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		fn         func(Service, *fakeClient) error
		wantErr    bool
		errContain string
	}{
		{
			name: "create success",
			fn: func(svc Service, _ *fakeClient) error {
				rec, err := svc.CreateRecord(context.Background(), CreateRecordInput{
					TableID: "t1", Fields: map[string]any{"Name": "a"},
				})
				if err != nil {
					return err
				}
				assert.Equal(t, "9", rec.ID)
				return nil
			},
		},
		{
			name: "create missing fields",
			fn: func(svc Service, _ *fakeClient) error {
				_, err := svc.CreateRecord(context.Background(), CreateRecordInput{TableID: "t1"})
				return err
			},
			wantErr:    true,
			errContain: "fields are required",
		},
		{
			name: "update injects id",
			fn: func(svc Service, fc *fakeClient) error {
				_, err := svc.UpdateRecord(context.Background(), UpdateRecordInput{
					TableID: "t1", RecordID: "7", Fields: map[string]any{"Name": "x"},
				})
				if err != nil {
					return err
				}
				assert.Equal(t, "7", fc.updateBody["Id"])
				assert.Equal(t, "x", fc.updateBody["Name"])
				return nil
			},
		},
		{
			name: "delete success",
			fn: func(svc Service, fc *fakeClient) error {
				err := svc.DeleteRecord(context.Background(), DeleteRecordInput{TableID: "t1", RecordID: "1"})
				if err != nil {
					return err
				}
				assert.Equal(t, "1", fc.deleteID)
				return nil
			},
		},
		{
			name: "get missing record id",
			fn: func(svc Service, _ *fakeClient) error {
				_, err := svc.GetRecord(context.Background(), GetRecordInput{TableID: "t1"})
				return err
			},
			wantErr:    true,
			errContain: "record_id is required",
		},
		{
			name: "list negative limit",
			fn: func(svc Service, _ *fakeClient) error {
				_, err := svc.ListRecords(context.Background(), ListRecordsInput{TableID: "t1", Limit: -1})
				return err
			},
			wantErr:    true,
			errContain: "limit must be non-negative",
		},
		{
			name: "not found maps ErrNotFound",
			fn: func(svc Service, _ *fakeClient) error {
				_, err := svc.GetRecord(context.Background(), GetRecordInput{TableID: "t1", RecordID: "9"})
				return err
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fc := &fakeClient{
				createResp: provider.Record{"Id": 9, "Name": "a"},
				updateResp: provider.Record{"Id": 7, "Name": "x"},
				record:     provider.Record{"Id": 1, "Name": "a"},
				records:    &provider.RecordList{List: []provider.Record{{"Id": 1}}},
			}
			if tt.name == "not found maps ErrNotFound" {
				fc.recordErr = &provider.StatusError{Op: "get record", Status: http.StatusNotFound, Msg: "missing"}
			}
			svc := NewWithClient(fc)
			err := tt.fn(svc, fc)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				if tt.name == "not found maps ErrNotFound" {
					require.ErrorIs(t, err, types.ErrNotFound)
				}
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestAdapter_HealthCheck(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		client  *fakeClient
		wantOK  bool
		wantErr bool
	}{
		{name: "healthy", client: &fakeClient{bases: &provider.BaseList{}}, wantOK: true},
		{name: "unhealthy", client: &fakeClient{basesErr: assert.AnError}, wantOK: false},
		{name: "canceled context", client: &fakeClient{}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := NewWithClient(tt.client)
			ctx := context.Background()
			if tt.name == "canceled context" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			ok, err := svc.HealthCheck(ctx)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantOK, ok)
		})
	}
}

func TestToRecord(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		input  provider.Record
		wantID string
	}{
		{name: "numeric id", input: provider.Record{"Id": float64(3), "Title": "x"}, wantID: "3"},
		{name: "string id", input: provider.Record{"Id": "abc", "Title": "x"}, wantID: "abc"},
		{name: "nil record", input: nil, wantID: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := toRecord(tt.input)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantID, got.ID)
			if tt.input != nil {
				assert.Equal(t, "x", got.Fields["Title"])
				_, hasID := got.Fields["Id"]
				assert.False(t, hasID)
			}
		})
	}
}

func TestNew_NilWhenUnconfigured(t *testing.T) {
	t.Parallel()
	assert.Nil(t, New())
	_ = capability.NocoRecord{}
}
