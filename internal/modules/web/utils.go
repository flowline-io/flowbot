package web

import (
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/route"
)

func renderError(ctx fiber.Ctx, msg string) error {
	ctx.Type("html")
	_, err := ctx.WriteString(`<div class="text-red-500 text-sm py-2">` + msg + `</div>`)
	return err
}

func getUID(ctx fiber.Ctx) string {
	rc := route.GetRequestContext(ctx)
	if rc == nil {
		return ""
	}
	return rc.UID.String()
}
