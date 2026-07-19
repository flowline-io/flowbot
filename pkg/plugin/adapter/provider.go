package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/plugin"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/types"
)

// PluginProviderAdapter implements provider OAuth and webhook interfaces.
type PluginProviderAdapter struct {
	runner plugin.Runner
	name   string
}

// NewProviderAdapter creates a provider adapter.
func NewProviderAdapter(r plugin.Runner, name string) *PluginProviderAdapter {
	return &PluginProviderAdapter{runner: r, name: name}
}

// WebhookConvert converts provider webhook payloads to DataEvents.
func (a *PluginProviderAdapter) WebhookConvert(payload []byte) ([]types.DataEvent, error) {
	raw, err := sonic.Marshal(map[string]any{"payload": payload})
	if err != nil {
		return nil, fmt.Errorf("webhook convert marshal: %w", err)
	}
	result, err := a.runner.Call(context.Background(), "webhook_convert", raw)
	if err != nil {
		return nil, fmt.Errorf("webhook convert: %w", err)
	}
	var events []types.DataEvent
	if err := sonic.Unmarshal(result, &events); err != nil {
		return nil, fmt.Errorf("webhook convert unmarshal: %w", err)
	}
	// Ensure non-nil Data/Tags for each event
	for i := range events {
		if events[i].Data == nil {
			events[i].Data = types.KV{}
		}
		if events[i].Tags == nil {
			events[i].Tags = types.KV{}
		}
		if events[i].CreatedAt.IsZero() {
			events[i].CreatedAt = time.Now()
		}
	}
	return events, nil
}

// GetAccessToken exchanges an authorization code for an access token.
// Remote plugin providers delegate token exchange to the plugin runner;
// stub providers return an unimplemented error.
func (*PluginProviderAdapter) GetAccessToken(_ fiber.Ctx) (*providers.OAuthToken, error) {
	return nil, fmt.Errorf("plugin provider adapter: GetAccessToken not implemented via remote plugin")
}

// GetAuthorizeURL returns the OAuth authorize URL from the plugin.
func (a *PluginProviderAdapter) GetAuthorizeURL(state string) string {
	raw, _ := sonic.Marshal(map[string]string{"state": state})
	result, err := a.runner.Call(context.Background(), "oauth_authorize", raw)
	if err != nil {
		return ""
	}
	var resp struct {
		URL string `json:"url"`
	}
	if err := sonic.Unmarshal(result, &resp); err != nil {
		return ""
	}
	return resp.URL
}
