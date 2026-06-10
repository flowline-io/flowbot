package ability

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

// WebhookHandler returns a Fiber handler that dispatches incoming webhook requests
// to the registered WebhookConverter for the given path.
func (m *EventSourceManager) WebhookHandler() fiber.Handler {
	return func(c fiber.Ctx) error {
		path := c.Params("*")
		if path == "" {
			return c.SendStatus(fiber.StatusNotFound)
		}

		m.mu.RLock()
		converter, ok := m.webhooks[path]
		m.mu.RUnlock()
		if !ok {
			return c.SendStatus(fiber.StatusNotFound)
		}

		body := c.Body()

		headers := make(map[string]string)
		c.Request().Header.VisitAll(func(key, value []byte) {
			headers[http.CanonicalHeaderKey(string(key))] = string(value)
		})
		c.Request().URI().QueryArgs().VisitAll(func(key, value []byte) {
			headers[http.CanonicalHeaderKey("X-Query-"+string(key))] = string(value)
		})

		if err := converter.VerifySignature(headers, body); err != nil {
			flog.Warn("event_source: webhook %s signature failed: %v", path, err)
			return c.SendStatus(fiber.StatusUnauthorized)
		}

		events, err := converter.Convert(body, headers)
		if err != nil {
			flog.Warn("event_source: webhook %s convert failed: %v", path, err)
			return c.SendStatus(fiber.StatusBadRequest)
		}

		sanitizedHeaders := sanitizeEventSourceHeaders(headers)
		webhookMethod := string(c.Request().Header.Method())
		webhookPath := string(c.Request().URI().Path())

		if m.metrics != nil {
			m.metrics.IncWebhookTotal(path, "202")
			m.metrics.IncWebhookEvents(path)
		}

		for _, ev := range events {
			if ev.Data == nil {
				ev.Data = make(types.KV)
			}
			ev.Data["_webhook_method"] = webhookMethod
			ev.Data["_webhook_path"] = webhookPath
			ev.Data["_webhook_status"] = 202
			ev.Data["_webhook_headers"] = sanitizedHeaders
			ev.Data["_webhook_body"] = truncateBody(body)
			if len(body) > maxWebhookBodySize {
				ev.Data["_webhook_body_truncated"] = true
			}

			m.poolSubmit(func() {
				if m.emitter != nil {
					if err := m.emitter(context.Background(), []types.DataEvent{ev}); err != nil {
						flog.Error(fmt.Errorf("event_source: webhook %s emit failed: %w", path, err))
					}
				}
			})
		}

		return c.SendStatus(fiber.StatusAccepted)
	}
}

// poolSubmit submits a function to the event pool, falling back to direct execution.
func (m *EventSourceManager) poolSubmit(fn func()) {
	if m.pool != nil {
		_ = m.pool.Invoke(fn)
	} else {
		fn()
	}
}

// eventSourceSensitiveHeaders lists headers that must be stripped before recording webhook metadata.
var eventSourceSensitiveHeaders = map[string]bool{
	"Authorization":       true,
	"Cookie":              true,
	"Set-Cookie":          true,
	"X-Api-Key":           true,
	"X-Hub-Signature":     true,
	"X-Hub-Signature-256": true,
	"X-Hmac-Signature":    true,
	"X-Webhook-Token":     true,
	"X-Gitlab-Token":      true,
	"X-Gogs-Signature":    true,
}

// sanitizeEventSourceHeaders removes sensitive headers from the request headers map.
func sanitizeEventSourceHeaders(headers map[string]string) map[string]string {
	out := make(map[string]string, len(headers))
	for k, v := range headers {
		if eventSourceSensitiveHeaders[k] {
			continue
		}
		out[k] = v
	}
	return out
}

// maxWebhookBodySize is the maximum webhook body size to store in DataEvent metadata.
const maxWebhookBodySize = 64 * 1024 // 64KB

// truncateBody truncates a body to maxWebhookBodySize for storage.
func truncateBody(body []byte) string {
	if len(body) <= maxWebhookBodySize {
		return string(body)
	}
	return string(body[:maxWebhookBodySize])
}
