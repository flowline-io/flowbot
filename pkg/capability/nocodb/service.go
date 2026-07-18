package nocodb

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/capability"
)

// ListTablesInput holds parameters for listing tables in a base.
type ListTablesInput struct {
	BaseID string
}

// GetTableInput holds parameters for fetching table metadata.
type GetTableInput struct {
	TableID string
}

// ListRecordsInput holds parameters for listing records.
type ListRecordsInput struct {
	TableID string
	Limit   int
	Offset  int
	Where   string
	Sort    string
	Fields  string
}

// GetRecordInput holds parameters for fetching a single record.
type GetRecordInput struct {
	TableID  string
	RecordID string
}

// CreateRecordInput holds parameters for creating a record.
type CreateRecordInput struct {
	TableID string
	Fields  map[string]any
}

// UpdateRecordInput holds parameters for updating a record.
type UpdateRecordInput struct {
	TableID  string
	RecordID string
	Fields   map[string]any
}

// DeleteRecordInput holds parameters for deleting a record.
type DeleteRecordInput struct {
	TableID  string
	RecordID string
}

// Service defines the NocoDB capability contract.
// List operations return the first page from NocoDB; use Page for pagination metadata.
// Record IDs assume the default NocoDB "Id" primary key (numeric or string custom PK).
type Service interface {
	ListBases(ctx context.Context) (*capability.ListResult[capability.NocoBase], error)
	ListTables(ctx context.Context, in ListTablesInput) (*capability.ListResult[capability.NocoTable], error)
	GetTable(ctx context.Context, in GetTableInput) (*capability.NocoTable, error)
	ListRecords(ctx context.Context, in ListRecordsInput) (*capability.ListResult[capability.NocoRecord], error)
	GetRecord(ctx context.Context, in GetRecordInput) (*capability.NocoRecord, error)
	CreateRecord(ctx context.Context, in CreateRecordInput) (*capability.NocoRecord, error)
	UpdateRecord(ctx context.Context, in UpdateRecordInput) (*capability.NocoRecord, error)
	DeleteRecord(ctx context.Context, in DeleteRecordInput) error
	HealthCheck(ctx context.Context) (bool, error)
}
