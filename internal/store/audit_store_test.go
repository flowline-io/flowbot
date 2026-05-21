package store

import (
	"context"
	"errors"
	"testing"

	"github.com/flowline-io/flowbot/pkg/audit"
	"github.com/stretchr/testify/assert"
)

func TestAuditStore_ImplementsAuditor(t *testing.T) {
	t.Parallel()
	var _ audit.Auditor = (*AuditStore)(nil)
}

func TestAuditStore_NilSafe(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		store *AuditStore
	}{
		{name: "nil store", store: nil},
		{name: "zero-value store with nil client", store: &AuditStore{}},
		{name: "zero-value store", store: &AuditStore{client: nil}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.store.Record(context.Background(), audit.Entry{
				Action: "test.action",
				Target: audit.Target{Type: "test", ID: "1"},
			})
			assert.NoError(t, err)
		})
	}
}

func TestAuditStore_RecordSuccess(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		store   *AuditStore
		entry   audit.Entry
		wantErr bool
	}{
		{name: "nil store", store: nil, entry: audit.Entry{Action: "test.success"}, wantErr: false},
		{name: "zero-value store", store: &AuditStore{}, entry: audit.Entry{Action: "test.success"}, wantErr: false},
		{name: "nil client", store: &AuditStore{client: nil}, entry: audit.Entry{Action: "test.success"}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.store.RecordSuccess(context.Background(), tt.entry)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAuditStore_RecordFailure(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		store   *AuditStore
		entry   audit.Entry
		err     error
		wantErr bool
	}{
		{name: "nil store with error", store: nil, entry: audit.Entry{Action: "test.fail"}, err: errors.New("boom"), wantErr: false},
		{name: "zero store with error", store: &AuditStore{}, entry: audit.Entry{Action: "test.fail"}, err: errors.New("boom"), wantErr: false},
		{name: "nil client with nil error", store: &AuditStore{client: nil}, entry: audit.Entry{Action: "test.fail"}, err: nil, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.store.RecordFailure(context.Background(), tt.entry, tt.err)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAuditStore_RecordRejected(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		store   *AuditStore
		entry   audit.Entry
		reason  string
		wantErr bool
	}{
		{name: "nil store", store: nil, entry: audit.Entry{Action: "test.deny"}, reason: "no permission", wantErr: false},
		{name: "zero store", store: &AuditStore{}, entry: audit.Entry{Action: "test.deny"}, reason: "no permission", wantErr: false},
		{name: "nil client", store: &AuditStore{client: nil}, entry: audit.Entry{Action: "test.deny"}, reason: "blocked", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.store.RecordRejected(context.Background(), tt.entry, tt.reason)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAuditStore_SubjectExtraction(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		subject *audit.Subject
	}{
		{
			name: "full subject with user details",
			subject: &audit.Subject{
				SubjectType: "user",
				SubjectID:   "owner",
				UID:         "auth0|123",
				IPAddress:   "10.0.0.1",
				UserAgent:   "test/1.0",
			},
		},
		{
			name:    "nil subject",
			subject: nil,
		},
		{
			name: "system pipeline subject",
			subject: &audit.Subject{
				SubjectType: "pipeline",
				SubjectID:   "system:pipeline",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := &AuditStore{}
			err := store.Record(context.Background(), audit.Entry{
				Subject: tt.subject,
				Action:  "test.action",
				Target:  audit.Target{Type: "test", ID: "1"},
			})
			assert.NoError(t, err)
		})
	}
}
