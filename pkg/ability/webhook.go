package ability

import (
	"context"
	"fmt"

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
			headers[string(key)] = string(value)
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

		if m.metrics != nil {
			m.metrics.IncWebhookTotal(path, "202")
			m.metrics.IncWebhookEvents(path)
		}

		for _, ev := range events {
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
