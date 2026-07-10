package web

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
)

func streamWebSessionEvents(ctx fiber.Ctx, sessionID string) error {
	sessionID = strings.Clone(sessionID)
	ctx.Set("Content-Type", "text/event-stream")
	ctx.Set("Cache-Control", "no-cache")
	ctx.Set("Connection", "keep-alive")

	reqCtx := ctx.Context()
	return ctx.SendStreamWriter(func(w *bufio.Writer) {
		hub := chatagent.GetSessionEventHub(sessionID)
		subID := fmt.Sprintf("web-%p", w)
		publisher := hub.Subscribe(subID, 32)
		defer hub.Unsubscribe(subID)

		sse := &chatagent.BufioSSEWriter{W: w}
		for {
			select {
			case <-reqCtx.Done():
				return
			case ev, ok := <-publisher.Events():
				if !ok {
					return
				}
				switch ev.Type {
				case chatagent.EventTypeConfirm, chatagent.EventTypeConfirmResolved, chatagent.EventTypeCanceled, chatagent.EventTypeModeChange:
					if sse.WriteEvent(ev) {
						return
					}
				}
			}
		}
	})
}
