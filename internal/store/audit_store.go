// Package store provides database storage implementations.
package store

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/pkg/audit"
	"github.com/flowline-io/flowbot/pkg/flog"
)

type AuditStore struct {
	client *gen.Client
}

func NewAuditStore(client *gen.Client) *AuditStore {
	return &AuditStore{client: client}
}

// Record writes an audit entry to persistent storage.
// If the store or client is nil, the call is silently skipped.
// Audit write failures are logged and do not propagate to the caller.
func (s *AuditStore) Record(ctx context.Context, entry audit.Entry) error {
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
	if entry.Request != nil {
		details["request"] = entry.Request
	}
	now := time.Now()
	_, err := s.client.AuditLog.Create().
		SetAction(entry.Action).
		SetTargetType(entry.Target.Type).
		SetTargetID(entry.Target.ID).
		SetActorUID(actorUID).
		SetDetails(details).
		SetCreatedAt(now).
		Save(ctx)
	if err != nil {
		flog.Warn("audit write failed: %v", err)
		return nil
	}
	return nil
}

// RecordSuccess writes a success audit entry.
func (s *AuditStore) RecordSuccess(ctx context.Context, entry audit.Entry) error {
	e := entry
	e.Request = wrapResult(entry.Request, "result", "success")
	return s.Record(ctx, e)
}

// RecordFailure writes a failure audit entry with the error message.
func (s *AuditStore) RecordFailure(ctx context.Context, entry audit.Entry, err error) error {
	e := entry
	e.Request = wrapResult(entry.Request, "result", "failed")
	if err != nil {
		e.Request = wrapResult(e.Request, "error", err.Error())
	}
	return s.Record(ctx, e)
}

// RecordRejected writes a rejected audit entry with the reason.
func (s *AuditStore) RecordRejected(ctx context.Context, entry audit.Entry, reason string) error {
	e := entry
	e.Request = wrapResult(entry.Request, "result", "rejected")
	e.Request = wrapResult(e.Request, "error", reason)
	return s.Record(ctx, e)
}

func wrapResult(request any, key, value string) map[string]any {
	m := map[string]any{key: value}
	if request != nil {
		if existing, ok := request.(map[string]any); ok {
			for k, v := range existing {
				if _, exists := m[k]; !exists {
					m[k] = v
				}
			}
		}
	}
	return m
}
