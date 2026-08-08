package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/flowline-io/flowbot/pkg/capability"
)

// EmailClient provides access to the email capability API.
type EmailClient struct {
	c *Client
}

// SendEmailRequest is the request body for sending email.
type SendEmailRequest struct {
	To       []string `json:"to"`
	Cc       []string `json:"cc,omitempty"`
	Bcc      []string `json:"bcc,omitempty"`
	Subject  string   `json:"subject"`
	Text     string   `json:"text,omitempty"`
	HTML     string   `json:"html,omitempty"`
	FromName string   `json:"from_name,omitempty"`
}

// EmailSearchRequest is the query for searching email.
type EmailSearchRequest struct {
	Mailbox string
	From    string
	To      string
	Subject string
	Since   string
	Before  string
	Unseen  *bool
	Limit   int
	Cursor  string
}

// EmailIDRequest carries a message id.
type EmailIDRequest struct {
	ID string `json:"id"`
}

// EmailPage holds pagination metadata.
type EmailPage struct {
	Limit      int    `json:"limit"`
	HasMore    bool   `json:"has_more"`
	NextCursor string `json:"next_cursor,omitzero"`
}

// EmailListResult holds messages and page info from InvokeResult.
type EmailListResult struct {
	Items []*capability.MailMessage `json:"data"`
	Page  EmailPage                 `json:"page"`
}

type emailItemResult struct {
	Item capability.MailMessage `json:"data"`
}

type emailActionResult struct {
	Data map[string]any `json:"data"`
}

type emailHealthResult struct {
	Healthy bool `json:"data"`
}

// Send sends an email.
func (e *EmailClient) Send(ctx context.Context, req *SendEmailRequest) error {
	if req == nil {
		return fmt.Errorf("request is required")
	}
	var result emailActionResult
	return e.c.Post(ctx, "/service/email/send", req, &result)
}

// ListMessages lists messages with optional filters.
func (e *EmailClient) ListMessages(ctx context.Context, mailbox string, unseenOnly *bool, limit int, cursor string) (*EmailListResult, error) {
	q := url.Values{}
	if mailbox != "" {
		q.Set("mailbox", mailbox)
	}
	if unseenOnly != nil {
		if *unseenOnly {
			q.Set("unseen_only", "true")
		} else {
			q.Set("unseen_only", "false")
		}
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	path := "/service/email/messages"
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	var result EmailListResult
	if err := e.c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetMessage fetches a message by id.
func (e *EmailClient) GetMessage(ctx context.Context, id string) (*capability.MailMessage, error) {
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	path := "/service/email/message?id=" + url.QueryEscape(id)
	var result emailItemResult
	if err := e.c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result.Item, nil
}

// SearchMessages searches messages.
func (e *EmailClient) SearchMessages(ctx context.Context, req *EmailSearchRequest) (*EmailListResult, error) {
	if req == nil {
		req = &EmailSearchRequest{}
	}
	q := url.Values{}
	if req.Mailbox != "" {
		q.Set("mailbox", req.Mailbox)
	}
	if req.From != "" {
		q.Set("from", req.From)
	}
	if req.To != "" {
		q.Set("to", req.To)
	}
	if req.Subject != "" {
		q.Set("subject", req.Subject)
	}
	if req.Since != "" {
		q.Set("since", req.Since)
	}
	if req.Before != "" {
		q.Set("before", req.Before)
	}
	if req.Unseen != nil {
		if *req.Unseen {
			q.Set("unseen", "true")
		} else {
			q.Set("unseen", "false")
		}
	}
	if req.Limit > 0 {
		q.Set("limit", strconv.Itoa(req.Limit))
	}
	if req.Cursor != "" {
		q.Set("cursor", req.Cursor)
	}
	path := "/service/email/search"
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	var result EmailListResult
	if err := e.c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// MarkRead marks a message as read.
func (e *EmailClient) MarkRead(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("id is required")
	}
	var result emailActionResult
	return e.c.Post(ctx, "/service/email/messages/read", &EmailIDRequest{ID: id}, &result)
}

// MarkUnread marks a message as unread.
func (e *EmailClient) MarkUnread(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("id is required")
	}
	var result emailActionResult
	return e.c.Post(ctx, "/service/email/messages/unread", &EmailIDRequest{ID: id}, &result)
}

// Health checks email backend connectivity.
func (e *EmailClient) Health(ctx context.Context) (bool, error) {
	var result emailHealthResult
	if err := e.c.Get(ctx, "/service/email/health", &result); err != nil {
		return false, err
	}
	return result.Healthy, nil
}
