package web

import (
	"context"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

func renderError(ctx fiber.Ctx, msg string) error {
	ctx.Type("html")
	return partials.FormError(msg).Render(context.Background(), ctx.Response().BodyWriter())
}

func getUID(ctx fiber.Ctx) string {
	rc := route.GetRequestContext(ctx)
	if rc == nil {
		return ""
	}
	return rc.UID.String()
}

// decodePathParam decodes a percent-encoded URL path parameter.
func decodePathParam(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	return url.PathUnescape(raw)
}

// pipelineNameParam returns the decoded :name path parameter for pipeline routes.
func pipelineNameParam(c fiber.Ctx) (string, error) {
	name, err := decodePathParam(c.Params("name"))
	if err != nil {
		return "", types.Errorf(types.ErrInvalidArgument, "invalid pipeline name")
	}
	return name, nil
}
