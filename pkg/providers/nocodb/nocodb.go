// Package nocodb implements the NocoDB provider.
package nocodb

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"resty.dev/v3"

	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/utils"
)

const (
	ID          = "nocodb"
	EndpointKey = "endpoint"
	TokenKey    = "token"

	maxErrorBodyLen = 200
	maxListLimit    = 1000
)

// StatusError is returned when NocoDB responds with a non-2xx status.
type StatusError struct {
	Op     string
	Status int
	Msg    string
}

// Error implements the error interface.
func (e *StatusError) Error() string {
	if e == nil {
		return "nocodb status error"
	}
	if e.Msg == "" {
		return fmt.Sprintf("%s: status %d", e.Op, e.Status)
	}
	return fmt.Sprintf("%s: status %d: %s", e.Op, e.Status, e.Msg)
}

// NocoDB is an HTTP client for the NocoDB v2 Data and Meta APIs.
type NocoDB struct {
	c *resty.Client
}

// GetClient builds a NocoDB client from vendors.nocodb config.
// Returns nil when endpoint is not configured.
func GetClient() *NocoDB {
	endpoint, _ := providers.GetConfig(ID, EndpointKey)
	token, _ := providers.GetConfig(ID, TokenKey)
	if endpoint.String() == "" {
		return nil
	}
	return NewNocoDB(endpoint.String(), token.String())
}

// NewNocoDB creates a NocoDB client with the given endpoint and API token.
// Authentication uses the xc-token header. Returns nil when endpoint is empty.
func NewNocoDB(endpoint, token string) *NocoDB {
	if endpoint == "" {
		return nil
	}
	v := &NocoDB{}
	v.c = utils.DefaultRestyClient()
	v.c.SetBaseURL(endpoint)
	v.c.SetHeader("xc-token", token)
	// NocoDB record delete sends a JSON body; resty v3 disables DELETE payloads by default.
	v.c.SetMethodDeleteAllowPayload(true)
	return v
}

// ListBases returns all bases visible to the API token (first page).
func (n *NocoDB) ListBases(ctx context.Context) (*BaseList, error) {
	var result BaseList
	resp, err := n.c.R().SetContext(ctx).SetResult(&result).Get("/api/v2/meta/bases")
	if err != nil {
		return nil, fmt.Errorf("list bases: %w", err)
	}
	if err := checkStatus(resp, "list bases"); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListTables returns tables belonging to the given base (first page).
func (n *NocoDB) ListTables(ctx context.Context, baseID string) (*TableList, error) {
	if err := validatePathID("base id", baseID); err != nil {
		return nil, err
	}
	var result TableList
	path := "/api/v2/meta/bases/" + url.PathEscape(baseID) + "/tables"
	resp, err := n.c.R().SetContext(ctx).SetResult(&result).Get(path)
	if err != nil {
		return nil, fmt.Errorf("list tables: %w", err)
	}
	if err := checkStatus(resp, "list tables"); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetTable returns table metadata including columns.
func (n *NocoDB) GetTable(ctx context.Context, tableID string) (*Table, error) {
	if err := validatePathID("table id", tableID); err != nil {
		return nil, err
	}
	var result Table
	path := "/api/v2/meta/tables/" + url.PathEscape(tableID)
	resp, err := n.c.R().SetContext(ctx).SetResult(&result).Get(path)
	if err != nil {
		return nil, fmt.Errorf("get table: %w", err)
	}
	if err := checkStatus(resp, "get table"); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListRecords returns records for a table with optional filters.
func (n *NocoDB) ListRecords(ctx context.Context, tableID string, q ListRecordsQuery) (*RecordList, error) {
	if err := validatePathID("table id", tableID); err != nil {
		return nil, err
	}
	if err := validateListQuery(q); err != nil {
		return nil, err
	}
	req := n.c.R().SetContext(ctx)
	if q.Limit > 0 {
		req.SetQueryParam("limit", strconv.Itoa(q.Limit))
	}
	if q.Offset > 0 {
		req.SetQueryParam("offset", strconv.Itoa(q.Offset))
	}
	if q.Where != "" {
		req.SetQueryParam("where", q.Where)
	}
	if q.Sort != "" {
		req.SetQueryParam("sort", q.Sort)
	}
	if q.Fields != "" {
		req.SetQueryParam("fields", q.Fields)
	}
	var result RecordList
	path := "/api/v2/tables/" + url.PathEscape(tableID) + "/records"
	resp, err := req.SetResult(&result).Get(path)
	if err != nil {
		return nil, fmt.Errorf("list records: %w", err)
	}
	if err := checkStatus(resp, "list records"); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetRecord returns a single record by ID.
func (n *NocoDB) GetRecord(ctx context.Context, tableID, recordID string) (Record, error) {
	if err := validatePathID("table id", tableID); err != nil {
		return nil, err
	}
	if err := validatePathID("record id", recordID); err != nil {
		return nil, err
	}
	var result Record
	path := "/api/v2/tables/" + url.PathEscape(tableID) + "/records/" + url.PathEscape(recordID)
	resp, err := n.c.R().SetContext(ctx).SetResult(&result).Get(path)
	if err != nil {
		return nil, fmt.Errorf("get record: %w", err)
	}
	if err := checkStatus(resp, "get record"); err != nil {
		return nil, err
	}
	return result, nil
}

// CreateRecord creates a record with the given field values.
func (n *NocoDB) CreateRecord(ctx context.Context, tableID string, fields map[string]any) (Record, error) {
	if err := validatePathID("table id", tableID); err != nil {
		return nil, err
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf("fields are required")
	}
	path := "/api/v2/tables/" + url.PathEscape(tableID) + "/records"
	resp, err := n.c.R().SetContext(ctx).SetBody(fields).Post(path)
	if err != nil {
		return nil, fmt.Errorf("create record: %w", err)
	}
	if err := checkStatus(resp, "create record"); err != nil {
		return nil, err
	}
	return decodeRecordBody(resp.Bytes())
}

// UpdateRecord updates a record identified by Id in fields.
// Numeric record IDs are sent as JSON numbers to match the NocoDB v2 contract.
func (n *NocoDB) UpdateRecord(ctx context.Context, tableID string, fields map[string]any) (Record, error) {
	if err := validatePathID("table id", tableID); err != nil {
		return nil, err
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf("fields are required")
	}
	body, err := withEncodedRecordID(fields)
	if err != nil {
		return nil, err
	}
	path := "/api/v2/tables/" + url.PathEscape(tableID) + "/records"
	resp, err := n.c.R().SetContext(ctx).SetBody(body).Patch(path)
	if err != nil {
		return nil, fmt.Errorf("update record: %w", err)
	}
	if err := checkStatus(resp, "update record"); err != nil {
		return nil, err
	}
	return decodeRecordBody(resp.Bytes())
}

// DeleteRecord deletes a record by ID.
// Numeric record IDs are sent as JSON numbers to match the NocoDB v2 contract.
func (n *NocoDB) DeleteRecord(ctx context.Context, tableID, id string) error {
	if err := validatePathID("table id", tableID); err != nil {
		return err
	}
	idVal, err := encodeRecordID(id)
	if err != nil {
		return err
	}
	body := []map[string]any{{"Id": idVal}}
	path := "/api/v2/tables/" + url.PathEscape(tableID) + "/records"
	resp, err := n.c.R().SetContext(ctx).SetBody(body).Delete(path)
	if err != nil {
		return fmt.Errorf("delete record: %w", err)
	}
	return checkStatus(resp, "delete record")
}

func checkStatus(resp *resty.Response, op string) error {
	if resp == nil {
		return fmt.Errorf("%s: empty response", op)
	}
	if resp.StatusCode() >= http.StatusOK && resp.StatusCode() < http.StatusMultipleChoices {
		return nil
	}
	msg := truncateMsg(string(resp.Bytes()))
	if msg == "" {
		msg = resp.Status()
	}
	return &StatusError{Op: op, Status: resp.StatusCode(), Msg: msg}
}

func truncateMsg(msg string) string {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return ""
	}
	if utf8.RuneCountInString(msg) <= maxErrorBodyLen {
		return msg
	}
	runes := []rune(msg)
	return string(runes[:maxErrorBodyLen]) + "..."
}

func decodeRecordBody(body []byte) (Record, error) {
	if len(body) == 0 {
		return nil, fmt.Errorf("empty response body")
	}
	trimmed := body
	for len(trimmed) > 0 && (trimmed[0] == ' ' || trimmed[0] == '\n' || trimmed[0] == '\t' || trimmed[0] == '\r') {
		trimmed = trimmed[1:]
	}
	if len(trimmed) > 0 && trimmed[0] == '[' {
		var list []Record
		if err := sonic.Unmarshal(body, &list); err != nil {
			return nil, fmt.Errorf("decode record: %w", err)
		}
		if len(list) == 0 {
			return nil, fmt.Errorf("empty record list in response")
		}
		return list[0], nil
	}
	var single Record
	if err := sonic.Unmarshal(body, &single); err != nil {
		return nil, fmt.Errorf("decode record: %w", err)
	}
	if single == nil || len(single) == 0 {
		return nil, fmt.Errorf("empty record in response")
	}
	return single, nil
}

// encodeRecordID converts a record id to the JSON value NocoDB expects.
// Decimal integers are encoded as int64; other ids remain strings (custom PKs).
func encodeRecordID(id string) (any, error) {
	if id == "" {
		return nil, fmt.Errorf("record id is required")
	}
	if n, err := strconv.ParseInt(id, 10, 64); err == nil {
		return n, nil
	}
	return id, nil
}

func withEncodedRecordID(fields map[string]any) (map[string]any, error) {
	id := recordID(fields)
	idVal, err := encodeRecordID(id)
	if err != nil {
		return nil, err
	}
	out := make(map[string]any, len(fields)+1)
	for k, v := range fields {
		switch k {
		case "Id", "id", "ID":
			continue
		default:
			out[k] = v
		}
	}
	out["Id"] = idVal
	return out, nil
}

func recordID(fields map[string]any) string {
	if fields == nil {
		return ""
	}
	for _, key := range []string{"Id", "id", "ID"} {
		if v, ok := fields[key]; ok {
			return formatAnyID(v)
		}
	}
	return ""
}

func formatAnyID(v any) string {
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

func validatePathID(name, id string) error {
	if id == "" {
		return fmt.Errorf("%s is required", name)
	}
	if strings.ContainsAny(id, "/?#") {
		return fmt.Errorf("%s contains invalid characters", name)
	}
	return nil
}

func validateListQuery(q ListRecordsQuery) error {
	if q.Limit < 0 {
		return fmt.Errorf("limit must be non-negative")
	}
	if q.Limit > maxListLimit {
		return fmt.Errorf("limit exceeds maximum of %d", maxListLimit)
	}
	if q.Offset < 0 {
		return fmt.Errorf("offset must be non-negative")
	}
	return nil
}

// AsStatusError extracts a StatusError from err if present.
func AsStatusError(err error) (*StatusError, bool) {
	var se *StatusError
	if errors.As(err, &se) {
		return se, true
	}
	return nil, false
}
