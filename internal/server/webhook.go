package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/pipeline"
	"github.com/flowline-io/flowbot/pkg/types"
)

// sensitiveHeaders are HTTP header names that must not be captured in webhook
// event data. All comparisons are case-insensitive.
var sensitiveHeaders = map[string]struct{}{
	"authorization":         {},
	"x-webhook-token":       {},
	"x-hub-signature":       {},
	"x-hub-signature-256":   {},
	"x-accesstoken":         {},
	"cookie":                {},
	"set-cookie":            {},
	"x-api-key":             {},
	"proxy-authorization":   {},
}

// sanitizeWebhookHeaders returns a copy of the request headers with sensitive
// headers removed. The wcfg auth header names are also excluded.
func sanitizeWebhookHeaders(c fiber.Ctx, wcfg *pipeline.WebhookConfig) map[string]string {
	headers := make(map[string]string)
	c.Request().Header.VisitAll(func(key, value []byte) {
		canonical := http.CanonicalHeaderKey(string(key))
		if _, sensitive := sensitiveHeaders[strings.ToLower(canonical)]; sensitive {
			return
		}
		if wcfg != nil {
			if wcfg.Auth.TokenHeader != "" && strings.ToLower(canonical) == strings.ToLower(wcfg.Auth.TokenHeader) {
				return
			}
			if wcfg.Auth.HMACHeader != "" && strings.ToLower(canonical) == strings.ToLower(wcfg.Auth.HMACHeader) {
				return
			}
		}
		headers[canonical] = string(value)
	})
	return headers
}

// registerWebhookRoutes registers webhook HTTP routes on the Fiber app
// for each webhook-enabled pipeline definition.
func registerWebhookRoutes(engine *pipeline.Engine) error {
	webhookMap, err := engine.RegisterWebhooks()
	if err != nil {
		return fmt.Errorf("register webhooks: %w", err)
	}

	for path, def := range webhookMap {
		method := def.Trigger.Webhook.Method
		routePath := "/webhook/" + strings.TrimPrefix(path, "/")
		handler := makeWebhookHandler(engine, def)
		switch method {
		case "GET":
			sharedAppPtr().Get(routePath, handler)
		case "POST":
			sharedAppPtr().Post(routePath, handler)
		case "PUT":
			sharedAppPtr().Put(routePath, handler)
		default:
			flog.Warn("webhook pipeline %s: unsupported method %q, skipping route registration", def.Name, method)
			continue
		}
		flog.Info("webhook route registered: %s %s -> pipeline %s", method, routePath, def.Name)
	}

	return nil
}

// makeWebhookHandler returns a Fiber handler that authenticates the request
// and dispatches to the engine.
func makeWebhookHandler(engine *pipeline.Engine, def *pipeline.Definition) fiber.Handler {
	return func(c fiber.Ctx) error {
		if def == nil || def.Trigger.Webhook == nil {
			return c.SendStatus(fiber.StatusNotFound)
		}

		wcfg := def.Trigger.Webhook

		status, ok := authenticateWebhook(c, wcfg)
		if !ok {
			return c.Status(status).SendString(http.StatusText(status))
		}

		eventID := types.Id()
		dataEvent := types.DataEvent{
			EventID:   eventID,
			EventType: wcfg.EventType,
			Source:    "webhook",
		}

		headers := sanitizeWebhookHeaders(c, wcfg)

		body := c.Body()

		if wcfg.Payload == config.WebhookPayloadMapped {
			var parsed map[string]any
			if err := sonic.Unmarshal(body, &parsed); err != nil {
				flog.Warn("webhook %s: invalid JSON for mapped payload", def.Name)
				return c.Status(fiber.StatusBadRequest).
					SendString("invalid JSON body")
			}
			dataEvent.Data = types.KV(parsed)
		} else {
			dataEvent.Data = make(types.KV)
			dataEvent.Data["_webhook_body"] = string(body)
		}

		dataEvent.Data["_webhook_headers"] = headers

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			if err := engine.ExecuteWebhook(ctx, def, dataEvent); err != nil {
				flog.Error(fmt.Errorf("webhook pipeline %s: %w", def.Name, err))
			}
		}()

		return c.SendStatus(fiber.StatusAccepted)
	}
}

// authenticateWebhook validates the request against the webhook auth config.
func authenticateWebhook(c fiber.Ctx, wcfg *pipeline.WebhookConfig) (int, bool) {
	if wcfg == nil {
		return fiber.StatusUnauthorized, false
	}
	ac := wcfg.Auth

	if ac.Token == "" && ac.HMACSecret == "" {
		return fiber.StatusUnauthorized, false
	}

	if ac.Token != "" {
		tokenHeader := ac.TokenHeader
		if tokenHeader == "" {
			tokenHeader = "X-Webhook-Token"
		}
		provided := c.Get(tokenHeader)
		if provided == ac.Token {
			return fiber.StatusOK, true
		}
	}

	if ac.HMACSecret != "" {
		hmacHeader := ac.HMACHeader
		if hmacHeader == "" {
			hmacHeader = "X-Hub-Signature-256"
		}
		provided := c.Get(hmacHeader)
		if verifyHMACSHA256(ac.HMACSecret, c.Body(), provided) {
			return fiber.StatusOK, true
		}
	}

	return fiber.StatusUnauthorized, false
}

// verifyHMACSHA256 verifies an HMAC-SHA256 signature against the body.
func verifyHMACSHA256(secret string, body []byte, signature string) bool {
	const prefix = "sha256="
	if !strings.HasPrefix(strings.ToLower(signature), prefix) {
		return false
	}
	expectedHex := strings.TrimPrefix(strings.ToLower(signature), prefix)
	expected, err := hex.DecodeString(expectedHex)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	actual := mac.Sum(nil)
	return hmac.Equal(actual, expected)
}
