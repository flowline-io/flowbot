package server

import (
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
	fbtrace "github.com/flowline-io/flowbot/pkg/trace"
	"github.com/flowline-io/flowbot/pkg/types"
)

// sensitiveHeaders are HTTP header names that must not be captured in webhook
// event data. All comparisons are case-insensitive.
var sensitiveHeaders = map[string]struct{}{
	"authorization":       {},
	"x-webhook-token":     {},
	"x-hub-signature":     {},
	"x-hub-signature-256": {},
	"x-accesstoken":       {},
	"cookie":              {},
	"set-cookie":          {},
	"x-api-key":           {},
	"proxy-authorization": {},
}

// sanitizeWebhookHeaders returns a copy of the request headers with sensitive
// headers removed. The wcfg auth header names are also excluded.
func sanitizeWebhookHeaders(c fiber.Ctx, wcfg *pipeline.WebhookConfig) map[string]string {
	headers := make(map[string]string)
	for key, value := range c.Request().Header.All() {
		canonical := http.CanonicalHeaderKey(string(key))
		if _, sensitive := sensitiveHeaders[strings.ToLower(canonical)]; sensitive {
			continue
		}
		if wcfg != nil {
			if wcfg.Auth.TokenHeader != "" && strings.EqualFold(canonical, wcfg.Auth.TokenHeader) {
				continue
			}
			if wcfg.Auth.HMACHeader != "" && strings.EqualFold(canonical, wcfg.Auth.HMACHeader) {
				continue
			}
		}
		headers[canonical] = string(value)
	}
	return headers
}

// registerWebhookRoutes mounts a catch-all handler under /webhook/*
// so ReloadDefinitions can update endpoints without re-registering Fiber routes.
// Reserved prefixes provider/ and workflow/ are ignored here; they have their own routes.
func registerWebhookRoutes(engine *pipeline.Engine) error {
	if _, err := engine.RegisterWebhooks(); err != nil {
		return fmt.Errorf("register webhooks: %w", err)
	}
	sharedAppPtr().All("/webhook/*", makePipelineWebhookCatchAll(engine))
	flog.Info("pipeline webhook route registered: ALL /webhook/*")
	return nil
}

func makePipelineWebhookCatchAll(engine *pipeline.Engine) fiber.Handler {
	return func(c fiber.Ctx) error {
		path := strings.TrimPrefix(c.Params("*"), "/")
		if path == "" || isReservedWebhookPrefix(path) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		def, ok := engine.LookupWebhook(path)
		if !ok || def == nil || def.Trigger.Webhook == nil {
			return c.SendStatus(fiber.StatusNotFound)
		}
		method := def.Trigger.Webhook.Method
		if method == "" {
			method = "POST"
		}
		if !strings.EqualFold(string(c.Request().Header.Method()), method) {
			return c.SendStatus(fiber.StatusMethodNotAllowed)
		}
		return makeWebhookHandler(engine, def)(c)
	}
}

func isReservedWebhookPrefix(path string) bool {
	path = strings.TrimPrefix(path, "/")
	return path == "provider" ||
		strings.HasPrefix(path, "provider/") ||
		path == "workflow" ||
		strings.HasPrefix(path, "workflow/")
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
			dataEvent.Data["_webhook_body"] = truncateWebhookBody(body)
			if len(body) > maxWebhookBodySize {
				dataEvent.Data["_webhook_body_truncated"] = true
			}
		}

		if dataEvent.Data == nil {
			dataEvent.Data = make(types.KV)
		}
		dataEvent.Data["_webhook_headers"] = headers
		dataEvent.Data["_webhook_method"] = string(c.Request().Header.Method())
		dataEvent.Data["_webhook_path"] = string(c.Request().URI().Path())
		dataEvent.Data["_webhook_status"] = fiber.StatusAccepted

		asyncCtx, asyncSpan := fbtrace.StartSpan(c.Context(), "pipeline.webhook.async")
		go func() {
			defer asyncSpan.End()
			ctx, cancel := fbtrace.DetachWithTimeout(asyncCtx, 10*time.Minute)
			defer cancel()
			if err := engine.ExecuteWebhook(ctx, def, dataEvent); err != nil {
				flog.Error(fmt.Errorf("webhook pipeline %s: %w", def.Name, err))
			}
		}()

		return c.SendStatus(fiber.StatusAccepted)
	}
}

// authenticateWebhook validates the request against the webhook auth config.
// Token auth accepts either the configured header (default X-Webhook-Token)
// or the query parameter "token".
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
		if provided == "" {
			provided = c.Query("token")
		}
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

// maxWebhookBodySize is the maximum webhook body size in bytes stored in events.
const maxWebhookBodySize = 64 * 1024

// truncateWebhookBody truncates the webhook body to maxWebhookBodySize bytes.
func truncateWebhookBody(body []byte) string {
	if len(body) <= maxWebhookBodySize {
		return string(body)
	}
	return string(body[:maxWebhookBodySize])
}
