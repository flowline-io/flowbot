package capability

import (
	"github.com/flowline-io/flowbot/pkg/types"
)

// VerifyQueryTokenWebhook validates a shared webhook token from X-Query-Token.
func VerifyQueryTokenWebhook(getToken func() string, headers map[string]string) error {
	token := getToken()
	if token == "" {
		return types.Errorf(types.ErrUnauthorized, "webhook token not configured")
	}
	provided, ok := headers["X-Query-Token"]
	if !ok {
		return types.Errorf(types.ErrUnauthorized, "missing token query parameter")
	}
	if provided != token {
		return types.Errorf(types.ErrUnauthorized, "invalid token")
	}
	return nil
}
