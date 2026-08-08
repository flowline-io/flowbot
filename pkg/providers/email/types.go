package email

import "time"

const (
	ID = "email"

	UsernameKey          = "username"
	PasswordKey          = "password"
	SMTPHostKey          = "smtp_host"
	SMTPPortKey          = "smtp_port"
	SMTPTLSKey           = "smtp_tls"
	IMAPHostKey          = "imap_host"
	IMAPPortKey          = "imap_port"
	IMAPTLSKey           = "imap_tls"
	MailboxKey           = "mailbox"
	UnseenOnlyKey        = "unseen_only"
	MarkSeenAfterEmitKey = "mark_seen_after_emit"

	TLSModeTLS      = "tls"
	TLSModeSTARTTLS = "starttls"
)

// Config holds SMTP/IMAP settings for the email provider.
type Config struct {
	Username          string
	Password          string
	SMTPHost          string
	SMTPPort          int
	SMTPTLS           string
	IMAPHost          string
	IMAPPort          int
	IMAPTLS           string
	Mailbox           string
	UnseenOnly        bool
	MarkSeenAfterEmit bool
}

// AttachmentMeta describes an attachment without binary content.
type AttachmentMeta struct {
	Filename string `json:"filename,omitzero"`
	MIMEType string `json:"mime_type,omitzero"`
	Size     int64  `json:"size,omitzero"`
}

// MessageMeta is email metadata returned by list/search/poller.
type MessageMeta struct {
	ID          string           `json:"id"`
	UID         uint32           `json:"uid"`
	UIDValidity uint32           `json:"uidvalidity"`
	Mailbox     string           `json:"mailbox"`
	From        []string         `json:"from,omitzero"`
	To          []string         `json:"to,omitzero"`
	Cc          []string         `json:"cc,omitzero"`
	Subject     string           `json:"subject,omitzero"`
	Date        time.Time        `json:"date,omitzero"`
	MessageID   string           `json:"message_id,omitzero"`
	Seen        bool             `json:"seen"`
	Attachments []AttachmentMeta `json:"attachments,omitzero"`
}

// Message is a full message including body text/html.
type Message struct {
	MessageMeta
	Text string `json:"text,omitzero"`
	HTML string `json:"html,omitzero"`
}

// SendInput holds outbound message fields.
type SendInput struct {
	To       []string
	Cc       []string
	Bcc      []string
	Subject  string
	Text     string
	HTML     string
	FromName string
}

// SearchQuery holds IMAP search filters and pagination.
type SearchQuery struct {
	Mailbox string
	From    string
	To      string
	Subject string
	Since   *time.Time
	Before  *time.Time
	Unseen  *bool
	Limit   int
	Cursor  string
}
