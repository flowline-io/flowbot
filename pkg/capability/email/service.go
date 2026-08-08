package email

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/pkg/capability"
)

// SendInput holds parameters for sending an email.
type SendInput struct {
	To       []string
	Cc       []string
	Bcc      []string
	Subject  string
	Text     string
	HTML     string
	FromName string
}

// ListInput holds parameters for listing messages.
type ListInput struct {
	Mailbox    string
	UnseenOnly *bool
	Page       capability.PageRequest
}

// SearchInput holds parameters for searching messages.
type SearchInput struct {
	Mailbox string
	From    string
	To      string
	Subject string
	Since   *time.Time
	Before  *time.Time
	Unseen  *bool
	Page    capability.PageRequest
}

// Service defines the email capability contract.
type Service interface {
	Send(ctx context.Context, in SendInput) error
	List(ctx context.Context, in ListInput) (*capability.ListResult[MailMessage], error)
	Get(ctx context.Context, id string) (*MailMessage, error)
	Search(ctx context.Context, in SearchInput) (*capability.ListResult[MailMessage], error)
	MarkRead(ctx context.Context, id string) error
	MarkUnread(ctx context.Context, id string) error
	HealthCheck(ctx context.Context) (bool, error)
	ListRawEvents(ctx context.Context, cursor string) ([]any, string, error)
	MarkEmittedSeen(ctx context.Context, ids []string) error
}
