// Package email implements the email capability adapter.
package email

import (
	"context"
	"time"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
	provider "github.com/flowline-io/flowbot/pkg/providers/email"
	"github.com/flowline-io/flowbot/pkg/types"
)

var emailCursorSecret = []byte("email-capability-cursor-v1")

// client is the provider subset used by the adapter.
type client interface {
	Send(ctx context.Context, in provider.SendInput) error
	List(ctx context.Context, mailbox string, unseenOnly bool, limit int, cursor string) ([]provider.MessageMeta, string, error)
	Search(ctx context.Context, q provider.SearchQuery) ([]provider.MessageMeta, string, error)
	Get(ctx context.Context, id string) (*provider.Message, error)
	MarkSeen(ctx context.Context, id string, seen bool) error
	HealthCheck(ctx context.Context) error
	ListRawEvents(ctx context.Context, cursor string) ([]map[string]any, string, error)
	Config() provider.Config
}

// Adapter implements Service using the email provider client.
type Adapter struct {
	client       client
	cursorSecret []byte
	now          func() time.Time
}

// New creates an Adapter from YAML config. Returns nil when not configured.
func New() Service {
	c, err := provider.GetClient()
	if err != nil || c == nil {
		return nil
	}
	return NewWithClient(c)
}

// NewWithClient creates an Adapter with a specific client (tests).
func NewWithClient(c client) Service {
	return &Adapter{
		client:       c,
		cursorSecret: emailCursorSecret,
		now:          time.Now,
	}
}

func (a *Adapter) Send(ctx context.Context, in SendInput) error {
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if len(in.To) == 0 {
		return types.Errorf(types.ErrInvalidArgument, "to is required")
	}
	if in.Subject == "" {
		return types.Errorf(types.ErrInvalidArgument, "subject is required")
	}
	if in.Text == "" && in.HTML == "" {
		return types.Errorf(types.ErrInvalidArgument, "text or html is required")
	}
	if err := a.client.Send(ctx, provider.SendInput{
		To: in.To, Cc: in.Cc, Bcc: in.Bcc,
		Subject: in.Subject, Text: in.Text, HTML: in.HTML, FromName: in.FromName,
	}); err != nil {
		return types.WrapError(types.ErrProvider, "email send failed", err)
	}
	return nil
}

func (a *Adapter) List(ctx context.Context, in ListInput) (*capability.ListResult[MailMessage], error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	unseen := a.client.Config().UnseenOnly
	if in.UnseenOnly != nil {
		unseen = *in.UnseenOnly
	}
	limit, providerCursor, err := a.decodePage(in.Page)
	if err != nil {
		return nil, err
	}
	metas, next, err := a.client.List(ctx, in.Mailbox, unseen, limit, providerCursor)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "email list failed", err)
	}
	return a.metaListResult(metas, next, limit)
}

func (a *Adapter) Get(ctx context.Context, id string) (*MailMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if id == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	msg, err := a.client.Get(ctx, id)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "email get failed", err)
	}
	return toMailMessage(msg), nil
}

func (a *Adapter) Search(ctx context.Context, in SearchInput) (*capability.ListResult[MailMessage], error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	limit, providerCursor, err := a.decodePage(in.Page)
	if err != nil {
		return nil, err
	}
	metas, next, err := a.client.Search(ctx, provider.SearchQuery{
		Mailbox: in.Mailbox,
		From:    in.From,
		To:      in.To,
		Subject: in.Subject,
		Since:   in.Since,
		Before:  in.Before,
		Unseen:  in.Unseen,
		Limit:   limit,
		Cursor:  providerCursor,
	})
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "email search failed", err)
	}
	return a.metaListResult(metas, next, limit)
}

func (a *Adapter) MarkRead(ctx context.Context, id string) error {
	return a.mark(ctx, id, true)
}

func (a *Adapter) MarkUnread(ctx context.Context, id string) error {
	return a.mark(ctx, id, false)
}

func (a *Adapter) mark(ctx context.Context, id string, seen bool) error {
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if id == "" {
		return types.Errorf(types.ErrInvalidArgument, "id is required")
	}
	if err := a.client.MarkSeen(ctx, id, seen); err != nil {
		return types.WrapError(types.ErrProvider, "email mark seen failed", err)
	}
	return nil
}

func (a *Adapter) HealthCheck(ctx context.Context) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if err := a.client.HealthCheck(ctx); err != nil {
		return false, types.WrapError(types.ErrProvider, "email health check failed", err)
	}
	return true, nil
}

func (a *Adapter) ListRawEvents(ctx context.Context, cursor string) ([]any, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	items, next, err := a.client.ListRawEvents(ctx, cursor)
	if err != nil {
		return nil, "", types.WrapError(types.ErrProvider, "email list raw events failed", err)
	}
	out := make([]any, len(items))
	for i, item := range items {
		out[i] = item
	}
	return out, next, nil
}

func (a *Adapter) MarkEmittedSeen(ctx context.Context, ids []string) error {
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if !a.client.Config().MarkSeenAfterEmit {
		return nil
	}
	for _, id := range ids {
		if id == "" {
			continue
		}
		if err := a.client.MarkSeen(ctx, id, true); err != nil {
			return types.WrapError(types.ErrProvider, "email mark seen after emit failed", err)
		}
	}
	return nil
}

func (a *Adapter) decodePage(page capability.PageRequest) (limit int, providerCursor string, err error) {
	limit = 20
	if page.Limit > 0 {
		limit = page.Limit
	}
	if page.Cursor == "" {
		return limit, "", nil
	}
	payload, err := capability.DecodeCursor(a.cursorSecret, page.Cursor, a.now())
	if err != nil {
		return 0, "", err
	}
	if payload.Capability != "" && payload.Capability != string(hub.CapEmail) {
		return 0, "", types.Errorf(types.ErrInvalidArgument, "invalid cursor capability")
	}
	if payload.Limit > 0 {
		limit = payload.Limit
	}
	return limit, payload.ProviderCursor, nil
}

func (a *Adapter) metaListResult(metas []provider.MessageMeta, next string, limit int) (*capability.ListResult[MailMessage], error) {
	items := make([]*MailMessage, 0, len(metas))
	for i := range metas {
		items = append(items, toMailMessageFromMeta(&metas[i]))
	}
	page := &capability.PageInfo{Limit: limit, HasMore: next != ""}
	if next != "" {
		cursor, err := capability.EncodeCursor(a.cursorSecret, capability.CursorPayload{
			Capability:     string(hub.CapEmail),
			Strategy:       "uid",
			ProviderCursor: next,
			Limit:          limit,
		})
		if err != nil {
			return nil, err
		}
		page.NextCursor = cursor
	}
	return &capability.ListResult[MailMessage]{Items: items, Page: page}, nil
}

func toMailMessage(m *provider.Message) *MailMessage {
	if m == nil {
		return nil
	}
	out := toMailMessageFromMeta(&m.MessageMeta)
	out.Text = m.Text
	out.HTML = m.HTML
	return out
}

func toMailMessageFromMeta(m *provider.MessageMeta) *MailMessage {
	out := &MailMessage{
		ID:        m.ID,
		Mailbox:   m.Mailbox,
		From:      append([]string(nil), m.From...),
		To:        append([]string(nil), m.To...),
		Cc:        append([]string(nil), m.Cc...),
		Subject:   m.Subject,
		Date:      m.Date,
		MessageID: m.MessageID,
		Seen:      m.Seen,
	}
	for _, a := range m.Attachments {
		out.Attachments = append(out.Attachments, MailAttachment{
			Filename: a.Filename,
			MIMEType: a.MIMEType,
			Size:     a.Size,
		})
	}
	return out
}

var _ Service = (*Adapter)(nil)
