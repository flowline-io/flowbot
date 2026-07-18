package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/flowline-io/flowbot/pkg/capability"
)

// NocodbClient provides access to the NocoDB capability API.
type NocodbClient struct {
	c *Client
}

// NocoCreateRecordRequest is the request body for creating a record.
type NocoCreateRecordRequest struct {
	Fields map[string]any `json:"fields"`
}

// NocoUpdateRecordRequest is the request body for updating a record.
type NocoUpdateRecordRequest struct {
	ID     string         `json:"id"`
	Fields map[string]any `json:"fields"`
}

// NocoDeleteRecordRequest is the request body for deleting a record.
type NocoDeleteRecordRequest struct {
	ID string `json:"id"`
}

// NocoListRecordsQuery holds optional query parameters for listing records.
type NocoListRecordsQuery struct {
	Limit  int
	Offset int
	Where  string
	Sort   string
	Fields string
}

const nocoMaxListLimit = 1000

// NocoBasesResult holds bases extracted from InvokeResult.
type NocoBasesResult struct {
	Items []*capability.NocoBase `json:"data"`
	Page  *capability.PageInfo   `json:"page"`
}

// NocoTablesResult holds tables extracted from InvokeResult.
type NocoTablesResult struct {
	Items []*capability.NocoTable `json:"data"`
	Page  *capability.PageInfo    `json:"page"`
}

// NocoTableResult holds a single table extracted from InvokeResult.
type NocoTableResult struct {
	Item capability.NocoTable `json:"data"`
}

// NocoRecordsResult holds records extracted from InvokeResult.
type NocoRecordsResult struct {
	Items []*capability.NocoRecord `json:"data"`
	Page  *capability.PageInfo     `json:"page"`
}

// NocoRecordResult holds a single record extracted from InvokeResult.
type NocoRecordResult struct {
	Item capability.NocoRecord `json:"data"`
}

// NocoDeleteResult holds delete confirmation extracted from InvokeResult.
type NocoDeleteResult struct {
	Data map[string]any `json:"data"`
}

// NocoHealthResult holds the health check result extracted from InvokeResult.
type NocoHealthResult struct {
	Healthy bool `json:"data"`
}

// ListBases returns NocoDB bases (first page).
func (n *NocodbClient) ListBases(ctx context.Context) (*NocoBasesResult, error) {
	var result NocoBasesResult
	if err := n.c.Get(ctx, "/service/nocodb/bases", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListTables returns tables in a base (first page).
func (n *NocodbClient) ListTables(ctx context.Context, baseID string) (*NocoTablesResult, error) {
	if baseID == "" {
		return nil, fmt.Errorf("base_id is required")
	}
	var result NocoTablesResult
	path := "/service/nocodb/bases/" + url.PathEscape(baseID) + "/tables"
	if err := n.c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetTable returns table metadata.
func (n *NocodbClient) GetTable(ctx context.Context, tableID string) (*capability.NocoTable, error) {
	if tableID == "" {
		return nil, fmt.Errorf("table_id is required")
	}
	var result NocoTableResult
	path := "/service/nocodb/tables/" + url.PathEscape(tableID)
	if err := n.c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result.Item, nil
}

// ListRecords returns records for a table.
func (n *NocodbClient) ListRecords(ctx context.Context, tableID string, q NocoListRecordsQuery) (*NocoRecordsResult, error) {
	if tableID == "" {
		return nil, fmt.Errorf("table_id is required")
	}
	if err := validateNocoListQuery(q); err != nil {
		return nil, err
	}
	values := url.Values{}
	if q.Limit > 0 {
		values.Set("limit", strconv.Itoa(q.Limit))
	}
	if q.Offset > 0 {
		values.Set("offset", strconv.Itoa(q.Offset))
	}
	if q.Where != "" {
		values.Set("where", q.Where)
	}
	if q.Sort != "" {
		values.Set("sort", q.Sort)
	}
	if q.Fields != "" {
		values.Set("fields", q.Fields)
	}
	path := "/service/nocodb/tables/" + url.PathEscape(tableID) + "/records"
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var result NocoRecordsResult
	if err := n.c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func validateNocoListQuery(q NocoListRecordsQuery) error {
	if q.Limit < 0 {
		return fmt.Errorf("limit must be non-negative")
	}
	if q.Limit > nocoMaxListLimit {
		return fmt.Errorf("limit exceeds maximum of %d", nocoMaxListLimit)
	}
	if q.Offset < 0 {
		return fmt.Errorf("offset must be non-negative")
	}
	return nil
}

// GetRecord returns a single record.
func (n *NocodbClient) GetRecord(ctx context.Context, tableID, recordID string) (*capability.NocoRecord, error) {
	if tableID == "" {
		return nil, fmt.Errorf("table_id is required")
	}
	if recordID == "" {
		return nil, fmt.Errorf("record_id is required")
	}
	var result NocoRecordResult
	path := "/service/nocodb/tables/" + url.PathEscape(tableID) + "/records/" + url.PathEscape(recordID)
	if err := n.c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result.Item, nil
}

// CreateRecord creates a record.
func (n *NocodbClient) CreateRecord(ctx context.Context, tableID string, fields map[string]any) (*capability.NocoRecord, error) {
	if tableID == "" {
		return nil, fmt.Errorf("table_id is required")
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf("fields are required")
	}
	var result NocoRecordResult
	path := "/service/nocodb/tables/" + url.PathEscape(tableID) + "/records"
	if err := n.c.Post(ctx, path, &NocoCreateRecordRequest{Fields: fields}, &result); err != nil {
		return nil, err
	}
	return &result.Item, nil
}

// UpdateRecord updates a record.
func (n *NocodbClient) UpdateRecord(ctx context.Context, tableID, recordID string, fields map[string]any) (*capability.NocoRecord, error) {
	if tableID == "" {
		return nil, fmt.Errorf("table_id is required")
	}
	if recordID == "" {
		return nil, fmt.Errorf("record_id is required")
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf("fields are required")
	}
	var result NocoRecordResult
	path := "/service/nocodb/tables/" + url.PathEscape(tableID) + "/records"
	if err := n.c.Patch(ctx, path, &NocoUpdateRecordRequest{ID: recordID, Fields: fields}, &result); err != nil {
		return nil, err
	}
	return &result.Item, nil
}

// DeleteRecord deletes a record.
func (n *NocodbClient) DeleteRecord(ctx context.Context, tableID, recordID string) error {
	if tableID == "" {
		return fmt.Errorf("table_id is required")
	}
	if recordID == "" {
		return fmt.Errorf("record_id is required")
	}
	var result NocoDeleteResult
	path := "/service/nocodb/tables/" + url.PathEscape(tableID) + "/records"
	return n.c.Delete(ctx, path, &NocoDeleteRecordRequest{ID: recordID}, &result)
}

// Health checks whether the NocoDB backend is reachable.
func (n *NocodbClient) Health(ctx context.Context) (bool, error) {
	var result NocoHealthResult
	if err := n.c.Get(ctx, "/service/nocodb/health", &result); err != nil {
		return false, err
	}
	return result.Healthy, nil
}
