package i18n

import (
	"context"

	"github.com/bytedance/sonic"
)

// Client message IDs exported to public/js via #flowbot-i18n.
var clientMessageIDs = []string{
	"common.confirm",
	"common.cancel",
	"confirm.default_title",
	"error.network",
	"error.timeout",
	"error.permission_denied",
	"error.validation",
	"error.not_found",
	"error.server",
	"error.request_failed",
	"error.try_again",
}

// ClientMessages returns localized client strings for #flowbot-i18n.
func ClientMessages(ctx context.Context) map[string]string {
	out := make(map[string]string, len(clientMessageIDs))
	for _, id := range clientMessageIDs {
		out[id] = T(ctx, id)
	}
	return out
}

// ClientJSON returns localized client strings as JSON bytes for templ embedding.
func ClientJSON(ctx context.Context) []byte {
	b, err := sonic.Marshal(ClientMessages(ctx))
	if err != nil {
		return []byte("{}")
	}
	return b
}

// ClientJSONString returns ClientJSON as a string for templ script tags.
func ClientJSONString(ctx context.Context) string {
	return string(ClientJSON(ctx))
}
