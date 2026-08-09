package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"slices"
	"strconv"
	"strings"

	"github.com/emersion/go-imap"
	imapclient "github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
)

const defaultListLimit = 20

// List returns message metadata for a mailbox with optional unseen filter and cursor pagination.
func (c *Client) List(ctx context.Context, mailbox string, unseenOnly bool, limit int, cursor string) ([]MessageMeta, string, error) {
	q := SearchQuery{
		Mailbox: mailbox,
		Limit:   limit,
		Cursor:  cursor,
	}
	if unseenOnly {
		v := true
		q.Unseen = &v
	}
	return c.Search(ctx, q)
}

// Search finds messages matching criteria.
func (c *Client) Search(ctx context.Context, q SearchQuery) ([]MessageMeta, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	if c.cfg.IMAPHost == "" {
		return nil, "", fmt.Errorf("email: imap_host is not configured")
	}
	mailbox, limit := normalizeSearchPaging(q, c.cfg.Mailbox)

	ic, err := c.dialIMAP(ctx)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = ic.Logout() }()

	mbox, err := ic.Select(mailbox, true)
	if err != nil {
		return nil, "", fmt.Errorf("email: select %s: %w", mailbox, err)
	}

	uids, err := ic.UidSearch(buildSearchCriteria(q))
	if err != nil {
		return nil, "", fmt.Errorf("email: search: %w", err)
	}
	if len(uids) == 0 {
		return []MessageMeta{}, "", nil
	}

	page, hasMore := pageUIDs(uids, q.Cursor, limit)
	metas, err := fetchMetas(ic, mailbox, mbox.UidValidity, page)
	if err != nil {
		return nil, "", err
	}
	return metas, nextUIDCursor(metas, hasMore), nil
}

func normalizeSearchPaging(q SearchQuery, defaultMailbox string) (mailbox string, limit int) {
	mailbox = q.Mailbox
	if mailbox == "" {
		mailbox = defaultMailbox
	}
	limit = q.Limit
	if limit <= 0 {
		limit = defaultListLimit
	}
	return mailbox, limit
}

func pageUIDs(uids []uint32, cursor string, limit int) ([]uint32, bool) {
	slices.SortFunc(uids, cmpUIDDesc)
	uids = applyUIDCursor(uids, cursor)
	if len(uids) > limit {
		return uids[:limit], true
	}
	return uids, false
}

func cmpUIDDesc(a, b uint32) int {
	switch {
	case a > b:
		return -1
	case a < b:
		return 1
	default:
		return 0
	}
}

func nextUIDCursor(metas []MessageMeta, hasMore bool) string {
	if !hasMore || len(metas) == 0 {
		return ""
	}
	return strconv.FormatUint(uint64(metas[len(metas)-1].UID), 10)
}

// Get fetches a single message by opaque id.
func (c *Client) Get(ctx context.Context, id string) (*Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ref, err := DecodeMessageID(id)
	if err != nil {
		return nil, err
	}
	mailbox := ref.Mailbox
	if mailbox == "" {
		mailbox = c.cfg.Mailbox
	}
	ic, err := c.dialIMAP(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = ic.Logout() }()

	mbox, err := ic.Select(mailbox, true)
	if err != nil {
		return nil, fmt.Errorf("email: select %s: %w", mailbox, err)
	}
	if mbox.UidValidity != ref.UIDValidity {
		return nil, fmt.Errorf("email: uidvalidity mismatch for id %s", id)
	}

	seq := new(imap.SeqSet)
	seq.AddNum(ref.UID)
	section := &imap.BodySectionName{}
	items := []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags, imap.FetchUid, section.FetchItem()}
	messages := make(chan *imap.Message, 1)
	done := make(chan error, 1)
	go func() {
		done <- ic.UidFetch(seq, items, messages)
	}()

	im := <-messages
	if err := <-done; err != nil {
		return nil, fmt.Errorf("email: fetch: %w", err)
	}
	if im == nil {
		return nil, fmt.Errorf("email: message %s not found", id)
	}

	meta := metaFromIMAP(mailbox, mbox.UidValidity, im)
	msg := &Message{MessageMeta: meta}
	if r := im.GetBody(section); r != nil {
		text, html, atts, err := parseBody(r)
		if err != nil {
			return nil, fmt.Errorf("email: parse body: %w", err)
		}
		msg.Text = text
		msg.HTML = html
		msg.Attachments = atts
	}
	return msg, nil
}

// MarkSeen sets or clears the \\Seen flag for a message id.
func (c *Client) MarkSeen(ctx context.Context, id string, seen bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	ref, err := DecodeMessageID(id)
	if err != nil {
		return err
	}
	mailbox := ref.Mailbox
	if mailbox == "" {
		mailbox = c.cfg.Mailbox
	}
	ic, err := c.dialIMAP(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = ic.Logout() }()

	mbox, err := ic.Select(mailbox, false)
	if err != nil {
		return fmt.Errorf("email: select %s: %w", mailbox, err)
	}
	if mbox.UidValidity != ref.UIDValidity {
		return fmt.Errorf("email: uidvalidity mismatch for id %s", id)
	}

	seq := new(imap.SeqSet)
	seq.AddNum(ref.UID)
	item := imap.FormatFlagsOp(imap.AddFlags, true)
	if !seen {
		item = imap.FormatFlagsOp(imap.RemoveFlags, true)
	}
	flags := []any{imap.SeenFlag}
	return ic.UidStore(seq, item, flags, nil)
}

// HealthCheck verifies IMAP login and SMTP handshake/auth when configured.
func (c *Client) HealthCheck(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if c.cfg.IMAPHost != "" {
		ic, err := c.dialIMAP(ctx)
		if err != nil {
			return err
		}
		_ = ic.Logout()
	}
	if c.cfg.SMTPHost != "" {
		if err := c.checkSMTP(ctx); err != nil {
			return err
		}
	}
	return nil
}

// ListRawEvents returns poller items (metadata maps) using configured mailbox/unseen defaults.
func (c *Client) ListRawEvents(ctx context.Context, cursor string) ([]map[string]any, string, error) {
	metas, next, err := c.List(ctx, c.cfg.Mailbox, c.cfg.UnseenOnly, defaultListLimit, cursor)
	if err != nil {
		return nil, "", err
	}
	items := make([]map[string]any, 0, len(metas))
	for _, m := range metas {
		items = append(items, metaToMap(m))
	}
	return items, next, nil
}

func (c *Client) dialIMAP(ctx context.Context) (*imapclient.Client, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	addr := net.JoinHostPort(c.cfg.IMAPHost, strconv.Itoa(c.cfg.IMAPPort))
	tlsConfig := &tls.Config{ServerName: c.cfg.IMAPHost, MinVersion: tls.VersionTLS12}

	var ic *imapclient.Client
	var err error
	switch c.cfg.IMAPTLS {
	case TLSModeTLS:
		ic, err = imapclient.DialTLS(addr, tlsConfig)
	default:
		ic, err = imapclient.Dial(addr)
		if err == nil {
			if err = ic.StartTLS(tlsConfig); err != nil {
				_ = ic.Logout()
				return nil, fmt.Errorf("email: imap starttls: %w", err)
			}
		}
	}
	if err != nil {
		return nil, fmt.Errorf("email: imap dial: %w", err)
	}
	if err := ic.Login(c.cfg.Username, c.cfg.Password); err != nil {
		_ = ic.Logout()
		return nil, fmt.Errorf("email: imap login: %w", err)
	}
	return ic, nil
}

func buildSearchCriteria(q SearchQuery) *imap.SearchCriteria {
	criteria := imap.NewSearchCriteria()
	if q.From != "" {
		criteria.Header.Add("From", q.From)
	}
	if q.To != "" {
		criteria.Header.Add("To", q.To)
	}
	if q.Subject != "" {
		criteria.Header.Add("Subject", q.Subject)
	}
	if q.Since != nil {
		criteria.Since = q.Since.UTC()
	}
	if q.Before != nil {
		criteria.Before = q.Before.UTC()
	}
	if q.Unseen != nil && *q.Unseen {
		criteria.WithoutFlags = []string{imap.SeenFlag}
	}
	if q.Unseen != nil && !*q.Unseen {
		criteria.WithFlags = []string{imap.SeenFlag}
	}
	return criteria
}

func applyUIDCursor(uids []uint32, cursor string) []uint32 {
	if cursor == "" {
		return uids
	}
	n, err := strconv.ParseUint(cursor, 10, 32)
	if err != nil {
		return uids
	}
	cut := uint32(n)
	out := make([]uint32, 0, len(uids))
	for _, u := range uids {
		if u < cut {
			out = append(out, u)
		}
	}
	return out
}

func fetchMetas(ic *imapclient.Client, mailbox string, uidValidity uint32, uids []uint32) ([]MessageMeta, error) {
	if len(uids) == 0 {
		return []MessageMeta{}, nil
	}
	seq := new(imap.SeqSet)
	seq.AddNum(uids...)
	items := []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags, imap.FetchUid, imap.FetchBodyStructure}
	messages := make(chan *imap.Message, len(uids))
	done := make(chan error, 1)
	go func() {
		done <- ic.UidFetch(seq, items, messages)
	}()

	byUID := make(map[uint32]*imap.Message, len(uids))
	for msg := range messages {
		if msg != nil {
			byUID[msg.Uid] = msg
		}
	}
	if err := <-done; err != nil {
		return nil, fmt.Errorf("email: fetch metas: %w", err)
	}

	out := make([]MessageMeta, 0, len(uids))
	for _, uid := range uids {
		im, ok := byUID[uid]
		if !ok {
			continue
		}
		out = append(out, metaFromIMAP(mailbox, uidValidity, im))
	}
	return out, nil
}

func metaFromIMAP(mailbox string, uidValidity uint32, im *imap.Message) MessageMeta {
	meta := MessageMeta{
		ID:          EncodeMessageID(mailbox, uidValidity, im.Uid),
		UID:         im.Uid,
		UIDValidity: uidValidity,
		Mailbox:     mailbox,
		Seen:        flagSeen(im.Flags),
	}
	if im.Envelope != nil {
		meta.Subject = im.Envelope.Subject
		meta.Date = im.Envelope.Date
		meta.MessageID = im.Envelope.MessageId
		meta.From = addressList(im.Envelope.From)
		meta.To = addressList(im.Envelope.To)
		meta.Cc = addressList(im.Envelope.Cc)
	}
	if im.BodyStructure != nil {
		meta.Attachments = attachmentMetas(im.BodyStructure)
	}
	return meta
}

func flagSeen(flags []string) bool {
	for _, f := range flags {
		if strings.EqualFold(f, imap.SeenFlag) {
			return true
		}
	}
	return false
}

func addressList(addrs []*imap.Address) []string {
	out := make([]string, 0, len(addrs))
	for _, a := range addrs {
		if a == nil {
			continue
		}
		out = append(out, a.Address())
	}
	return out
}

func attachmentMetas(bs *imap.BodyStructure) []AttachmentMeta {
	if bs == nil {
		return nil
	}
	var out []AttachmentMeta
	var walk func(b *imap.BodyStructure)
	walk = func(b *imap.BodyStructure) {
		if b == nil {
			return
		}
		disposition := strings.ToLower(b.Disposition)
		if disposition == "attachment" || (b.MIMEType != "text" && b.MIMEType != "multipart" && b.MIMEType != "") {
			filename := ""
			if b.DispositionParams != nil {
				filename = b.DispositionParams["filename"]
			}
			if filename == "" && b.Params != nil {
				filename = b.Params["name"]
			}
			if filename != "" || disposition == "attachment" {
				out = append(out, AttachmentMeta{
					Filename: filename,
					MIMEType: strings.ToLower(b.MIMEType + "/" + b.MIMESubType),
					Size:     int64(b.Size),
				})
			}
		}
		for _, part := range b.Parts {
			walk(part)
		}
	}
	walk(bs)
	return out
}

func parseBody(r io.Reader) (text, html string, atts []AttachmentMeta, err error) {
	mr, err := mail.CreateReader(r)
	if err != nil {
		return "", "", nil, err
	}
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return text, html, atts, err
		}
		switch h := p.Header.(type) {
		case *mail.InlineHeader:
			ct, _, _ := h.ContentType()
			body, readErr := io.ReadAll(p.Body)
			if readErr != nil {
				return text, html, atts, readErr
			}
			switch {
			case strings.HasPrefix(ct, "text/html"):
				html = string(body)
			case strings.HasPrefix(ct, "text/plain"):
				text = string(body)
			}
		case *mail.AttachmentHeader:
			filename, _ := h.Filename()
			ct, _, _ := h.ContentType()
			n, _ := io.Copy(io.Discard, p.Body)
			atts = append(atts, AttachmentMeta{
				Filename: filename,
				MIMEType: ct,
				Size:     n,
			})
		}
	}
	return text, html, atts, nil
}

func metaToMap(m MessageMeta) map[string]any {
	return map[string]any{
		"id":          m.ID,
		"mailbox":     m.Mailbox,
		"from":        m.From,
		"to":          m.To,
		"cc":          m.Cc,
		"subject":     m.Subject,
		"date":        m.Date,
		"message_id":  m.MessageID,
		"seen":        m.Seen,
		"attachments": m.Attachments,
	}
}
