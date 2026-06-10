package backoff

import (
	"context"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// Middleware returns a Watermill handler middleware that wraps h with retry
// behavior governed by cfg. Each retry is logged via logger.
// The cfg is copied per message to avoid data races on concurrent invocations.
func Middleware(cfg Config, logger watermill.LoggerAdapter) message.HandlerMiddleware {
	return func(h message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			producedMessages, err := h(msg)
			if err == nil {
				return producedMessages, nil
			}

			localCfg := cfg
			// Share adaptive state across retries for the same middleware instance.
			localCfg.adaptiveState = cfg.adaptiveState
			origOnRetry := localCfg.OnRetry
			localCfg.OnRetry = func(attempt int, delay time.Duration, err error) {
				if logger != nil {
					logger.Error("Retrying after error", err, watermill.LogFields{
						"retry_attempt": attempt,
						"retry_delay":   delay,
					})
				}
				if origOnRetry != nil {
					origOnRetry(attempt, delay, err)
				}
			}

			attempt, finalErr := Do(msg.Context(), localCfg, func(_ context.Context) error {
				producedMessages, err = h(msg)
				return err
			})
			if finalErr != nil {
				if logger != nil {
					logger.Error(fmt.Sprintf("All %d retry attempts exhausted", attempt), finalErr, watermill.LogFields{
						"final_attempt": attempt,
					})
				}
				return nil, finalErr
			}
			return producedMessages, nil
		}
	}
}
