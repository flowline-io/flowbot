package example

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
)

func TestWebserviceEndpoints(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{
			name:       "GET /example returns OK",
			method:     "GET",
			path:       "/service/example/example",
			wantStatus: 200,
		},
		{
			name:       "GET /get without id returns 400",
			method:     "GET",
			path:       "/service/example/get",
			wantStatus: 400,
		},
		{
			name:       "POST /webhook/example returns 202",
			method:     "POST",
			path:       "/service/example/webhook/example",
			wantStatus: 202,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New(fiber.Config{
				ErrorHandler: func(ctx fiber.Ctx, err error) error {
					code, _ := mapDomainErrorStatus(err)
					return ctx.Status(code).SendString(err.Error())
				},
			})
			ruleSets := append(webserviceRules, webhookRules...)
			module.Webservice(app, Name, webservice.Ruleset(ruleSets))

			req := httptest.NewRequest(tt.method, tt.path, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)

			_ = app.Shutdown()
		})
	}
}

func mapDomainErrorStatus(err error) (int, bool) {
	switch {
	case errors.Is(err, types.ErrInvalidArgument):
		return fiber.StatusBadRequest, true
	case errors.Is(err, types.ErrUnauthorized):
		return fiber.StatusUnauthorized, true
	case errors.Is(err, types.ErrForbidden):
		return fiber.StatusForbidden, true
	case errors.Is(err, types.ErrNotFound):
		return fiber.StatusNotFound, true
	case errors.Is(err, types.ErrAlreadyExists), errors.Is(err, types.ErrConflict):
		return fiber.StatusConflict, true
	case errors.Is(err, types.ErrRateLimited):
		return fiber.StatusTooManyRequests, true
	case errors.Is(err, types.ErrUnavailable):
		return fiber.StatusServiceUnavailable, true
	case errors.Is(err, types.ErrTimeout):
		return fiber.StatusGatewayTimeout, true
	case errors.Is(err, types.ErrNotImplemented):
		return fiber.StatusNotImplemented, true
	case errors.Is(err, types.ErrProvider):
		return fiber.StatusBadGateway, true
	case errors.Is(err, types.ErrInternal):
		return fiber.StatusInternalServerError, true
	default:
		return fiber.StatusInternalServerError, false
	}
}
