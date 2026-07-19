package web

import (
	"context"
	"net/url"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	"github.com/flowline-io/flowbot/pkg/route"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

// renderError writes a FormError HTML fragment without changing the HTMX swap target.
// Prefer renderFormError for form mutations or toastError for action-only failures.
func renderError(ctx fiber.Ctx, msg string) error {
	ctx.Type("html")
	return partials.FormError(msg).Render(context.Background(), ctx.Response().BodyWriter())
}

// renderFormError writes a FormError fragment and sets HX-Retarget / HX-Reswap so HTMX
// places it into cssTarget (typically "#form-error" with innerHTML).
func renderFormError(ctx fiber.Ctx, cssTarget, msg string) error {
	if cssTarget == "" {
		cssTarget = "#form-error"
	}
	ctx.Response().Header.Set("HX-Retarget", cssTarget)
	ctx.Response().Header.Set("HX-Reswap", "innerHTML")
	return renderError(ctx, msg)
}

// showToastTrigger builds an HX-Trigger payload for the web UI toast system.
func showToastTrigger(toastType, message string) (string, error) {
	payload, err := sonic.Marshal(map[string]any{
		"showToast": map[string]string{
			"type":    toastType,
			"message": message,
		},
	})
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

// setShowToast sets the HX-Trigger header so HTMX can fire a showToast event.
func setShowToast(ctx fiber.Ctx, toastType, message string) {
	trigger, err := showToastTrigger(toastType, message)
	if err != nil {
		return
	}
	ctx.Set("HX-Trigger", trigger)
}

// toastError sets an error toast via HX-Trigger and returns 204 so HTMX does not
// swap the current target (avoids wiping forms/tables on action-only failures).
func toastError(ctx fiber.Ctx, msg string) error {
	setShowToast(ctx, "error", msg)
	return ctx.SendStatus(fiber.StatusNoContent)
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
