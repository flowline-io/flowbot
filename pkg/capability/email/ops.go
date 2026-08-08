package email

import "github.com/flowline-io/flowbot/pkg/capability"

// MailMessage is the capability-facing email message type.
type MailMessage = capability.MailMessage

// MailAttachment describes an attachment without binary payload.
type MailAttachment = capability.MailAttachment

// Operation name constants for the email capability.
const (
	OpSend       = "send"
	OpList       = "list"
	OpGet        = "get"
	OpSearch     = "search"
	OpMarkRead   = "mark_read"
	OpMarkUnread = "mark_unread"
	OpHealth     = "health"
)
