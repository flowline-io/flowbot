package example

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"

	"github.com/bytedance/sonic"

	provider "github.com/flowline-io/flowbot/pkg/providers/example"
	"github.com/flowline-io/flowbot/pkg/types"
)

// ExampleWebhook implements capability.WebhookConverter for the example provider.
// It demonstrates signature verification and payload conversion patterns.
type ExampleWebhook struct {
	getSecret func() string
}

// NewExampleWebhook creates an ExampleWebhook that reads the HMAC secret
// from the example provider config at verification time.
func NewExampleWebhook() *ExampleWebhook {
	return &ExampleWebhook{
		getSecret: provider.GetWebhookSecret,
	}
}

// WebhookPath returns the URL path that receives webhook events from the example provider.
func (*ExampleWebhook) WebhookPath() string {
	return "example"
}

// VerifySignature validates the HMAC-SHA256 signature from the X-Signature header.
// The secret is read from provider config at verification time via getSecret.
func (w *ExampleWebhook) VerifySignature(headers map[string]string, body []byte) error {
	secret := w.getSecret()
	if secret == "" {
		return types.Errorf(types.ErrUnauthorized, "webhook secret not configured")
	}
	signature, ok := headers["X-Signature"]
	if !ok {
		return types.Errorf(types.ErrUnauthorized, "missing X-Signature header")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return types.Errorf(types.ErrUnauthorized, "invalid signature")
	}
	return nil
}

// Convert transforms the raw webhook body into one or more DataEvent records.
func (*ExampleWebhook) Convert(body []byte, _ map[string]string) ([]types.DataEvent, error) {
	var payload struct {
		EventType string `json:"event_type"`
		EntityID  string `json:"entity_id"`
		Data      any    `json:"data"`
	}
	if err := sonic.Unmarshal(body, &payload); err != nil {
		return nil, types.Errorf(types.ErrInvalidArgument, "invalid webhook payload: %v", err)
	}
	ev := types.DataEvent{
		EventID:        types.Id(),
		EventType:      payload.EventType,
		Source:         "example_webhook",
		EntityID:       payload.EntityID,
		IdempotencyKey: payload.EntityID,
		Data:           types.KV{"event": payload.Data},
	}
	return []types.DataEvent{ev}, nil
}
