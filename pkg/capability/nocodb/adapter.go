// Package nocodb implements the NocoDB adapter for the database capability.
package nocodb

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/flowline-io/flowbot/pkg/capability"
	provider "github.com/flowline-io/flowbot/pkg/providers/nocodb"
	"github.com/flowline-io/flowbot/pkg/types"
)

const maxListLimit = 1000

// client defines the subset of provider.NocoDB methods used by this adapter.
type client interface {
	ListBases(ctx context.Context) (*provider.BaseList, error)
	ListTables(ctx context.Context, baseID string) (*provider.TableList, error)
	GetTable(ctx context.Context, tableID string) (*provider.Table, error)
	ListRecords(ctx context.Context, tableID string, q provider.ListRecordsQuery) (*provider.RecordList, error)
	GetRecord(ctx context.Context, tableID, recordID string) (provider.Record, error)
	CreateRecord(ctx context.Context, tableID string, fields map[string]any) (provider.Record, error)
	UpdateRecord(ctx context.Context, tableID string, fields map[string]any) (provider.Record, error)
	DeleteRecord(ctx context.Context, tableID, id string) error
}

// Adapter implements Service using the NocoDB provider client.
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

// ListBases returns bases visible to the configured API token (first page).
func (a *Adapter) ListBases(ctx context.Context) (*capability.ListResult[capability.NocoBase], error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	result, err := a.client.ListBases(ctx)
	if err != nil {
		return nil, wrapProviderErr("nocodb list bases failed", err)
	}
	if result == nil {
		return &capability.ListResult[capability.NocoBase]{
			Items: []*capability.NocoBase{},
			Page:  &capability.PageInfo{},
		}, nil
	}
	out := make([]*capability.NocoBase, 0, len(result.List))
	for _, item := range result.List {
		out = append(out, toBase(item))
	}
	return &capability.ListResult[capability.NocoBase]{
		Items: out,
		Page:  toPageInfo(result.PageInfo, 0),
	}, nil
}

// ListTables returns tables in a base (first page).
func (a *Adapter) ListTables(ctx context.Context, in ListTablesInput) (*capability.ListResult[capability.NocoTable], error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if in.BaseID == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "base_id is required")
	}
	result, err := a.client.ListTables(ctx, in.BaseID)
	if err != nil {
		return nil, wrapProviderErr("nocodb list tables failed", err)
	}
	if result == nil {
		return &capability.ListResult[capability.NocoTable]{
			Items: []*capability.NocoTable{},
			Page:  &capability.PageInfo{},
		}, nil
	}
	out := make([]*capability.NocoTable, 0, len(result.List))
	for _, item := range result.List {
		out = append(out, toTable(item))
	}
	return &capability.ListResult[capability.NocoTable]{
		Items: out,
		Page:  toPageInfo(result.PageInfo, 0),
	}, nil
}

// GetTable returns table metadata including columns.
func (a *Adapter) GetTable(ctx context.Context, in GetTableInput) (*capability.NocoTable, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if in.TableID == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "table_id is required")
	}
	table, err := a.client.GetTable(ctx, in.TableID)
	if err != nil {
		return nil, wrapProviderErr("nocodb get table failed", err)
	}
	return toTablePtr(table), nil
}

// ListRecords returns records for a table.
func (a *Adapter) ListRecords(ctx context.Context, in ListRecordsInput) (*capability.ListResult[capability.NocoRecord], error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if in.TableID == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "table_id is required")
	}
	if err := validateListBounds(in.Limit, in.Offset); err != nil {
		return nil, err
	}
	result, err := a.client.ListRecords(ctx, in.TableID, provider.ListRecordsQuery{
		Limit:  in.Limit,
		Offset: in.Offset,
		Where:  in.Where,
		Sort:   in.Sort,
		Fields: in.Fields,
	})
	if err != nil {
		return nil, wrapProviderErr("nocodb list records failed", err)
	}
	if result == nil {
		return &capability.ListResult[capability.NocoRecord]{
			Items: []*capability.NocoRecord{},
			Page:  &capability.PageInfo{Limit: in.Limit},
		}, nil
	}
	out := make([]*capability.NocoRecord, 0, len(result.List))
	for _, item := range result.List {
		out = append(out, toRecord(item))
	}
	return &capability.ListResult[capability.NocoRecord]{
		Items: out,
		Page:  toPageInfo(result.PageInfo, in.Limit),
	}, nil
}

// GetRecord returns a single record.
func (a *Adapter) GetRecord(ctx context.Context, in GetRecordInput) (*capability.NocoRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if in.TableID == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "table_id is required")
	}
	if in.RecordID == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "record_id is required")
	}
	rec, err := a.client.GetRecord(ctx, in.TableID, in.RecordID)
	if err != nil {
		return nil, wrapProviderErr("nocodb get record failed", err)
	}
	return toRecord(rec), nil
}

// CreateRecord creates a record with the given fields.
func (a *Adapter) CreateRecord(ctx context.Context, in CreateRecordInput) (*capability.NocoRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if in.TableID == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "table_id is required")
	}
	if len(in.Fields) == 0 {
		return nil, types.Errorf(types.ErrInvalidArgument, "fields are required")
	}
	rec, err := a.client.CreateRecord(ctx, in.TableID, in.Fields)
	if err != nil {
		return nil, wrapProviderErr("nocodb create record failed", err)
	}
	return toRecord(rec), nil
}

// UpdateRecord updates a record by ID.
// Numeric IDs are encoded as JSON numbers by the provider.
func (a *Adapter) UpdateRecord(ctx context.Context, in UpdateRecordInput) (*capability.NocoRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if in.TableID == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "table_id is required")
	}
	if in.RecordID == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "record_id is required")
	}
	if len(in.Fields) == 0 {
		return nil, types.Errorf(types.ErrInvalidArgument, "fields are required")
	}
	body := make(map[string]any, len(in.Fields)+1)
	for k, v := range in.Fields {
		body[k] = v
	}
	body["Id"] = in.RecordID
	rec, err := a.client.UpdateRecord(ctx, in.TableID, body)
	if err != nil {
		return nil, wrapProviderErr("nocodb update record failed", err)
	}
	return toRecord(rec), nil
}

// DeleteRecord deletes a record by ID.
func (a *Adapter) DeleteRecord(ctx context.Context, in DeleteRecordInput) error {
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if in.TableID == "" {
		return types.Errorf(types.ErrInvalidArgument, "table_id is required")
	}
	if in.RecordID == "" {
		return types.Errorf(types.ErrInvalidArgument, "record_id is required")
	}
	if err := a.client.DeleteRecord(ctx, in.TableID, in.RecordID); err != nil {
		return wrapProviderErr("nocodb delete record failed", err)
	}
	return nil
}

// HealthCheck reports whether the NocoDB backend is reachable.
// Provider failures are treated as unhealthy (false, nil), not as invoke errors.
func (a *Adapter) HealthCheck(ctx context.Context) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	_, err := a.client.ListBases(ctx)
	if err != nil {
		return false, nil
	}
	return true, nil
}

func wrapProviderErr(msg string, err error) error {
	if se, ok := provider.AsStatusError(err); ok {
		switch se.Status {
		case http.StatusNotFound:
			return types.WrapError(types.ErrNotFound, msg, err)
		case http.StatusBadRequest, http.StatusUnprocessableEntity:
			return types.WrapError(types.ErrInvalidArgument, msg, err)
		case http.StatusUnauthorized, http.StatusForbidden:
			return types.WrapError(types.ErrForbidden, msg, err)
		}
	}
	return types.WrapError(types.ErrProvider, msg, err)
}

func validateListBounds(limit, offset int) error {
	if limit < 0 {
		return types.Errorf(types.ErrInvalidArgument, "limit must be non-negative")
	}
	if limit > maxListLimit {
		return types.Errorf(types.ErrInvalidArgument, "limit exceeds maximum of %d", maxListLimit)
	}
	if offset < 0 {
		return types.Errorf(types.ErrInvalidArgument, "offset must be non-negative")
	}
	return nil
}

func toPageInfo(p provider.PageInfo, fallbackLimit int) *capability.PageInfo {
	limit := p.PageSize
	if limit == 0 {
		limit = fallbackLimit
	}
	var total *int64
	if p.TotalRows > 0 || p.Page > 0 || p.PageSize > 0 {
		t := int64(p.TotalRows)
		total = &t
	}
	return &capability.PageInfo{
		Limit:   limit,
		HasMore: !p.IsLastPage && (p.PageSize > 0 || p.TotalRows > 0),
		Total:   total,
	}
}

func toBase(b provider.Base) *capability.NocoBase {
	return &capability.NocoBase{ID: b.ID, Title: b.Title}
}

func toTable(t provider.Table) *capability.NocoTable {
	out := &capability.NocoTable{ID: t.ID, Title: t.Title, BaseID: t.BaseID}
	if len(t.Columns) > 0 {
		out.Columns = make([]capability.NocoColumn, 0, len(t.Columns))
		for _, c := range t.Columns {
			out.Columns = append(out.Columns, capability.NocoColumn{
				ID:    c.ID,
				Title: c.Title,
				Type:  c.UIDT,
			})
		}
	}
	return out
}

func toTablePtr(t *provider.Table) *capability.NocoTable {
	if t == nil {
		return nil
	}
	return toTable(*t)
}

func toRecord(r provider.Record) *capability.NocoRecord {
	if r == nil {
		return &capability.NocoRecord{Fields: map[string]any{}}
	}
	fields := make(map[string]any, len(r))
	var id string
	for k, v := range r {
		switch k {
		case "Id", "id", "ID":
			id = formatID(v)
		default:
			fields[k] = v
		}
	}
	return &capability.NocoRecord{ID: id, Fields: fields}
}

func formatID(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatInt(int64(t), 10)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	default:
		return fmt.Sprint(t)
	}
}

// Compile-time interface check.
var _ Service = (*Adapter)(nil)
