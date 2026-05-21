// Package audit provides the Auditor interface and supporting types for audit logging.
package audit

import "context"

// Auditor writes audit entries to persistent storage.
type Auditor interface {
	Record(ctx context.Context, entry Entry) error
	RecordSuccess(ctx context.Context, entry Entry) error
	RecordFailure(ctx context.Context, entry Entry, err error) error
	RecordRejected(ctx context.Context, entry Entry, reason string) error
}

// Entry represents a single audit record.
type Entry struct {
	Subject *Subject
	Action  string
	Target  Target
	Request any
}

// Subject carries authenticated actor identity.
type Subject struct {
	SubjectType string
	SubjectID   string
	UID         string
	IPAddress   string
	UserAgent   string
}

// Target identifies the resource being acted on.
type Target struct {
	Type string
	ID   string
}
