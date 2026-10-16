// Package store provides database storage implementations.
package store

import (
	"context"
	"strconv"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqljson"
	"github.com/bytedance/sonic"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/auditlog"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/predicate"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/toolerr"
	"github.com/flowline-io/flowbot/pkg/types/audit"
)

// ---------------------------------------------------------------------------
// AuditStore
// ---------------------------------------------------------------------------

type AuditStore struct {
	client *gen.Client
}

func NewAuditStore(client *gen.Client) *AuditStore {
	return &AuditStore{client: client}
}

// AuditStoreFromDB returns an AuditStore using the global database client.
func AuditStoreFromDB() *AuditStore {
	return NewAuditStore(ClientFromDB())
}

const (
	auditResultSuccess  = "success"
	auditResultFailure  = "failure"
	auditResultRejected = "rejected"
	auditRequestMaxByte = 4096
)

// Record writes an audit entry to persistent storage.
// If the store or client is nil, the call is silently skipped.
// Audit write failures are logged and do not propagate to the caller.
// Sensitive fields in entry.Request are redacted before storage.
func (s *AuditStore) Record(ctx context.Context, entry audit.Entry) error {
	return s.writeEntry(ctx, entry, "", "")
}

// RecordSuccess writes a success audit entry.
func (s *AuditStore) RecordSuccess(ctx context.Context, entry audit.Entry) error {
	return s.writeEntry(ctx, entry, auditResultSuccess, "")
}

// RecordFailure writes a failure audit entry. The stored error_code is the tool error code.
func (s *AuditStore) RecordFailure(ctx context.Context, entry audit.Entry, err error) error {
	code := "internal"
	if err != nil {
		code = toolerr.ToToolError(err).Code
	}
	return s.writeEntry(ctx, entry, auditResultFailure, code)
}

// RecordRejected writes a rejected audit entry. A single error-code token is stored as error_code.
func (s *AuditStore) RecordRejected(ctx context.Context, entry audit.Entry, reason string) error {
	return s.writeEntry(ctx, entry, auditResultRejected, rejectedCode(reason))
}

func (s *AuditStore) writeEntry(ctx context.Context, entry audit.Entry, result, errorCode string) error {
	if s == nil || s.client == nil {
		return nil
	}
	actorUID := ""
	details := map[string]any{}
	if entry.Subject != nil {
		actorUID = entry.Subject.SubjectType + ":" + entry.Subject.SubjectID
		details["subject_type"] = entry.Subject.SubjectType
		details["subject_id"] = entry.Subject.SubjectID
		details["uid"] = entry.Subject.UID
		details["ip_address"] = entry.Subject.IPAddress
		details["user_agent"] = entry.Subject.UserAgent
	}
	if entry.CorrelationID != "" {
		details["correlation_id"] = entry.CorrelationID
	}
	if entry.Request != nil {
		details["request"] = sanitizeAuditValue(entry.Request)
	}
	now := time.Now()
	_, err := s.client.AuditLog.Create().
		SetAction(entry.Action).
		SetTargetType(entry.Target.Type).
		SetTargetID(entry.Target.ID).
		SetActorUID(actorUID).
		SetResult(result).
		SetErrorCode(errorCode).
		SetDetails(details).
		SetCreatedAt(now).
		Save(ctx)
	if err != nil {
		flog.Warn("audit write failed: %v", err)
		return nil
	}
	return nil
}

// AuditRecord is one audit row returned by List.
type AuditRecord struct {
	ID            int64
	At            time.Time
	Action        string
	SubjectType   string
	SubjectID     string
	TargetType    string
	TargetID      string
	Result        string
	ErrorCode     string
	Request       any
	Truncated     bool
	CorrelationID string
}

// ListAuditOptions filters audit rows. Limit defaults to 20 and caps at 100.
type ListAuditOptions struct {
	Action     string
	SubjectID  string
	TargetType string
	Result     string
	Since      time.Time
	Until      time.Time
	Limit      int
	Cursor     string
}

// List returns audit rows newest first. The cursor is the last row id.
// Rows are ordered by created_at and id descending; ids increase with inserts.
// SubjectID matches details.uid, the caller uid stored on the row.
func (s *AuditStore) List(ctx context.Context, opts ListAuditOptions) ([]AuditRecord, string, error) {
	if s == nil || s.client == nil {
		return nil, "", nil
	}
	limit := auditPageLimit(opts.Limit)
	q := s.client.AuditLog.Query()
	if opts.Action != "" {
		q = q.Where(auditlog.ActionEQ(opts.Action))
	}
	if opts.TargetType != "" {
		q = q.Where(auditlog.TargetTypeEQ(opts.TargetType))
	}
	if opts.Result != "" {
		q = q.Where(auditlog.ResultEQ(opts.Result))
	}
	if opts.SubjectID != "" {
		q = q.Where(detailsUIDEQ(opts.SubjectID))
	}
	if !opts.Since.IsZero() {
		q = q.Where(auditlog.CreatedAtGTE(opts.Since))
	}
	if !opts.Until.IsZero() {
		q = q.Where(auditlog.CreatedAtLTE(opts.Until))
	}
	if id, ok := parseAuditCursor(opts.Cursor); ok {
		q = q.Where(auditlog.IDLT(id))
	}
	rows, err := q.Order(auditlog.ByCreatedAt(sql.OrderDesc()), auditlog.ByID(sql.OrderDesc())).Limit(limit + 1).All(ctx)
	if err != nil {
		return nil, "", err
	}
	next := ""
	if len(rows) > limit {
		last := rows[limit-1]
		next = strconv.FormatInt(last.ID, 10)
		rows = rows[:limit]
	}
	out := make([]AuditRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, auditRecordFrom(row))
	}
	return out, next, nil
}

func auditPageLimit(n int) int {
	if n <= 0 {
		return 20
	}
	if n > 100 {
		return 100
	}
	return n
}

func detailsUIDEQ(uid string) predicate.AuditLog {
	return predicate.AuditLog(func(s *sql.Selector) {
		s.Where(sqljson.ValueEQ(s.C(auditlog.FieldDetails), uid, sqljson.Path("uid")))
	})
}

func detailString(details map[string]any, key string) string {
	value, ok := details[key].(string)
	if !ok {
		return ""
	}
	return value
}

func parseAuditCursor(cursor string) (int64, bool) {
	id, err := strconv.ParseInt(cursor, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func auditRecordFrom(row *gen.AuditLog) AuditRecord {
	rec := AuditRecord{
		ID:         row.ID,
		At:         row.CreatedAt,
		Action:     row.Action,
		TargetType: row.TargetType,
		TargetID:   row.TargetID,
		Result:     row.Result,
		ErrorCode:  row.ErrorCode,
	}
	if row.Details != nil {
		rec.SubjectType = detailString(row.Details, "subject_type")
		rec.SubjectID = detailString(row.Details, "subject_id")
		rec.CorrelationID = detailString(row.Details, "correlation_id")
		rec.Request = row.Details["request"]
		if raw, err := sonic.Marshal(rec.Request); err == nil && len(raw) > auditRequestMaxByte {
			rec.Request = map[string]any{"preview": string(raw[:auditRequestMaxByte])}
			rec.Truncated = true
		}
	}
	return rec
}

// Item is the list payload shared by REST and MCP.
func (r AuditRecord) Item() map[string]any {
	item := map[string]any{
		"id":     r.ID,
		"at":     r.At.UTC().Format(time.RFC3339Nano),
		"action": r.Action,
		"subject": map[string]any{
			"type": r.SubjectType,
			"id":   r.SubjectID,
		},
		"target": map[string]any{
			"type": r.TargetType,
			"id":   r.TargetID,
		},
		"result":         r.Result,
		"error_code":     r.ErrorCode,
		"request":        r.Request,
		"correlation_id": r.CorrelationID,
	}
	if r.Truncated {
		item["truncated"] = true
	}
	return item
}

func rejectedCode(reason string) string {
	if reason == "" {
		return auditResultRejected
	}
	for _, r := range reason {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' {
			return auditResultRejected
		}
	}
	return reason
}

// auditSensitiveKeys lists request field names that are redacted from audit
// logs. All comparisons are case-insensitive (lowercase).
var auditSensitiveKeys = map[string]struct{}{
	"password":      {},
	"passwd":        {},
	"secret":        {},
	"token":         {},
	"api_key":       {},
	"apikey":        {},
	"authorization": {},
	"cookie":        {},
	"private_key":   {},
	"access_token":  {},
	"auth":          {},
	"credential":    {},
	"signature":     {},
}

// sanitizeAuditValue recursively redacts sensitive fields from audit data.
// Map keys matching auditSensitiveKeys are replaced with "[redacted]".
// Non-map values are returned unchanged.
func sanitizeAuditValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, vv := range val {
			if _, sensitive := auditSensitiveKeys[strings.ToLower(k)]; sensitive {
				out[k] = "[redacted]"
				continue
			}
			out[k] = sanitizeAuditValue(vv)
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, vv := range val {
			out[i] = sanitizeAuditValue(vv)
		}
		return out
	default:
		return v
	}
}
