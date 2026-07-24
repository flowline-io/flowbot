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
	svc := chatAgentService()
	return ctx.SendStreamWriter(func(w *bufio.Writer) {
		hub := svc.GetSessionEventHub(sessionID)
		subID := fmt.Sprintf("web-%p", w)
		publisher := hub.Subscribe(subID, 32)
		defer hub.Unsubscribe(subID)

		sse := &chatagent.BufioSSEWriter{W: w}
		if svc.WritePendingConfirmIfAny(sessionID, func(ev chatagent.StreamEvent) bool {
			return sse.WriteEvent(ev)
		}) {
			return
		}

		for {
			select {
			case <-reqCtx.Done():
				return
			case ev, ok := <-publisher.Events():
				if !ok {
					return
				}
				if chatagent.IsObserverStreamEvent(ev.Type) {
					if sse.WriteEvent(ev) {
						return
					}
				}
			}
		}
	})
}
