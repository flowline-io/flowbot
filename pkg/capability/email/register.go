package email

import (
	"context"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
)

// Register registers the email capability with hub and invoker registry.
// When svc is nil the provider is not configured and registration is skipped.
func Register(app string, svc Service) error {
	return capability.Register(buildSpec(app, svc))
}

// CatalogSpec returns capability metadata for documentation.
func CatalogSpec() capability.Spec {
	return buildSpec("", nil)
}

func buildSpec(app string, svc Service) capability.Spec {
	return capability.Spec{
		Type:        hub.CapEmail,
		App:         app,
		Description: "Email capability for SMTP send and IMAP read",
		Instance:    svc,
		Events: []hub.EventDef{
			{Name: "email/messages.created", Description: "Fires when the poller observes a new message"},
			{Name: "email/messages.updated", Description: "Fires when the poller observes a changed message"},
		},
		Ops: []capability.OpDef{
			{
				Name: OpSend, Description: "Send an email", Scopes: []string{auth.ScopeServiceEmailWrite}, Mutation: true,
				Input: []hub.ParamDef{
					{Name: "to", Type: "[]string", Required: true, Description: "Recipient addresses"},
					{Name: "cc", Type: "[]string", Required: false, Description: "Cc addresses"},
					{Name: "bcc", Type: "[]string", Required: false, Description: "Bcc addresses"},
					{Name: "subject", Type: "string", Required: true, Description: "Subject line"},
					{Name: "text", Type: "string", Required: false, Description: "Plain text body"},
					{Name: "html", Type: "string", Required: false, Description: "HTML body"},
					{Name: "from_name", Type: "string", Required: false, Description: "Display name for From"},
				},
				Handler: invokeSend(svc),
			},
			{
				Name: OpList, Description: "List messages", Scopes: []string{auth.ScopeServiceEmailRead},
				Input: []hub.ParamDef{
					{Name: "mailbox", Type: "string", Required: false, Description: "Mailbox name (default INBOX)"},
					{Name: "unseen_only", Type: "bool", Required: false, Description: "Only unseen messages"},
					{Name: "limit", Type: "int", Required: false, Description: "Maximum items per page"},
					{Name: "cursor", Type: "string", Required: false, Description: "Opaque pagination cursor"},
				},
				Handler: invokeList(svc),
			},
			{
				Name: OpGet, Description: "Get a message by id", Scopes: []string{auth.ScopeServiceEmailRead},
				Input:   []hub.ParamDef{{Name: "id", Type: "string", Required: true, Description: "Opaque message id"}},
				Handler: invokeGet(svc),
			},
			{
				Name: OpSearch, Description: "Search messages", Scopes: []string{auth.ScopeServiceEmailRead},
				Input: []hub.ParamDef{
					{Name: "mailbox", Type: "string", Required: false, Description: "Mailbox name"},
					{Name: "from", Type: "string", Required: false, Description: "From filter"},
					{Name: "to", Type: "string", Required: false, Description: "To filter"},
					{Name: "subject", Type: "string", Required: false, Description: "Subject filter"},
					{Name: "since", Type: "string", Required: false, Description: "RFC3339 lower bound"},
					{Name: "before", Type: "string", Required: false, Description: "RFC3339 upper bound"},
					{Name: "unseen", Type: "bool", Required: false, Description: "Unseen filter"},
					{Name: "limit", Type: "int", Required: false, Description: "Maximum items per page"},
					{Name: "cursor", Type: "string", Required: false, Description: "Opaque pagination cursor"},
				},
				Handler: invokeSearch(svc),
			},
			{
				Name: OpMarkRead, Description: "Mark a message as read", Scopes: []string{auth.ScopeServiceEmailWrite}, Mutation: true,
				Input:   []hub.ParamDef{{Name: "id", Type: "string", Required: true, Description: "Opaque message id"}},
				Handler: invokeMark(svc, true),
			},
			{
				Name: OpMarkUnread, Description: "Mark a message as unread", Scopes: []string{auth.ScopeServiceEmailWrite}, Mutation: true,
				Input:   []hub.ParamDef{{Name: "id", Type: "string", Required: true, Description: "Opaque message id"}},
				Handler: invokeMark(svc, false),
			},
			{
				Name: OpHealth, Description: "Health check", Scopes: []string{auth.ScopeServiceEmailRead},
				Handler: invokeHealth(svc),
			},
		},
	}
}

func invokeSend(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		to, err := stringSliceParam(params, "to", true)
		if err != nil {
			return nil, err
		}
		subject, err := capability.RequiredString(params, "subject")
		if err != nil {
			return nil, err
		}
		text, _ := capability.StringParam(params, "text")
		html, _ := capability.StringParam(params, "html")
		fromName, _ := capability.StringParam(params, "from_name")
		cc, _ := stringSliceParam(params, "cc", false)
		bcc, _ := stringSliceParam(params, "bcc", false)
		if err := svc.Send(ctx, SendInput{
			To: to, Cc: cc, Bcc: bcc, Subject: subject, Text: text, HTML: html, FromName: fromName,
		}); err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: map[string]any{"sent": true}, Text: "email sent"}, nil
	}
}

func invokeList(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		mailbox, _ := capability.StringParam(params, "mailbox")
		var unseen *bool
		if b, ok := capability.BoolParam(params, "unseen_only"); ok {
			unseen = &b
		}
		result, err := svc.List(ctx, ListInput{
			Mailbox:    mailbox,
			UnseenOnly: unseen,
			Page:       capability.PageRequestFromParams(params),
		})
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &capability.ListResult[MailMessage]{Items: []*MailMessage{}, Page: &capability.PageInfo{}}
		}
		return &capability.InvokeResult{Data: result.Items, Page: result.Page}, nil
	}
}

func invokeGet(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		id, err := capability.RequiredString(params, "id")
		if err != nil {
			return nil, err
		}
		msg, err := svc.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: msg}, nil
	}
}

func invokeSearch(svc Service) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		mailbox, _ := capability.StringParam(params, "mailbox")
		from, _ := capability.StringParam(params, "from")
		to, _ := capability.StringParam(params, "to")
		subject, _ := capability.StringParam(params, "subject")
		since, err := optionalTime(params, "since")
		if err != nil {
			return nil, err
		}
		before, err := optionalTime(params, "before")
		if err != nil {
			return nil, err
		}
		var unseen *bool
		if b, ok := capability.BoolParam(params, "unseen"); ok {
			unseen = &b
		}
		result, err := svc.Search(ctx, SearchInput{
			Mailbox: mailbox, From: from, To: to, Subject: subject,
			Since: since, Before: before, Unseen: unseen,
			Page: capability.PageRequestFromParams(params),
		})
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = &capability.ListResult[MailMessage]{Items: []*MailMessage{}, Page: &capability.PageInfo{}}
		}
		return &capability.InvokeResult{Data: result.Items, Page: result.Page}, nil
	}
}

func invokeMark(svc Service, seen bool) capability.Invoker {
	return func(ctx context.Context, params map[string]any) (*capability.InvokeResult, error) {
		id, err := capability.RequiredString(params, "id")
		if err != nil {
			return nil, err
		}
		if seen {
			if err := svc.MarkRead(ctx, id); err != nil {
				return nil, err
			}
		} else {
			if err := svc.MarkUnread(ctx, id); err != nil {
				return nil, err
			}
		}
		return &capability.InvokeResult{Data: map[string]any{"id": id, "seen": seen}}, nil
	}
}

func invokeHealth(svc Service) capability.Invoker {
	return func(ctx context.Context, _ map[string]any) (*capability.InvokeResult, error) {
		ok, err := svc.HealthCheck(ctx)
		if err != nil {
			return nil, err
		}
		return &capability.InvokeResult{Data: ok}, nil
	}
}

func stringSliceParam(params map[string]any, key string, required bool) ([]string, error) {
	v, ok := params[key]
	if !ok || v == nil {
		if required {
			return nil, types.Errorf(types.ErrInvalidArgument, "%s is required", key)
		}
		return nil, nil
	}
	out, ok := normalizeStringList(v)
	if !ok {
		return nil, types.Errorf(types.ErrInvalidArgument, "%s must be a string list", key)
	}
	if required && len(out) == 0 {
		return nil, types.Errorf(types.ErrInvalidArgument, "%s is required", key)
	}
	return out, nil
}

func normalizeStringList(v any) ([]string, bool) {
	switch t := v.(type) {
	case []string:
		return t, true
	case string:
		parts := strings.Split(t, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		return out, true
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			s, ok := item.(string)
			if !ok {
				continue
			}
			s = strings.TrimSpace(s)
			if s != "" {
				out = append(out, s)
			}
		}
		return out, true
	default:
		return nil, false
	}
}

func optionalTime(params map[string]any, key string) (*time.Time, error) {
	s, ok := capability.StringParam(params, key)
	if !ok || s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, types.Errorf(types.ErrInvalidArgument, "%s must be RFC3339", key)
	}
	return &t, nil
}
