package hub

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/types"
)

func mapDomainErrors(err error) (int, bool) {
	switch {
	case errors.Is(err, types.ErrInvalidArgument):
		return fiber.StatusBadRequest, true
	case errors.Is(err, types.ErrNotFound):
		return fiber.StatusNotFound, true
	default:
		return fiber.StatusInternalServerError, false
	}
}

func errorHandler(ctx fiber.Ctx, err error) error {
	if err == nil {
		return nil
	}
	code, _ := mapDomainErrors(err)
	return ctx.Status(code).SendString(err.Error())
}

func TestQueryByTag_Validation(t *testing.T) {
	tests := []struct {
		name       string
		queryStr   string
		wantStatus int
	}{
		{"missing key returns 400", "value=alpha", 400},
		{"missing value returns 400", "key=project", 400},
		{"empty key and value returns 400", "key=&value=", 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New(fiber.Config{
				ErrorHandler: errorHandler,
			})
			app.Get("/resource-chain", queryByTag)
			defer app.Shutdown()
			req := httptest.NewRequest(fiber.MethodGet, "/resource-chain?"+tt.queryStr, nil)
			resp, _ := app.Test(req)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

func TestGetRelations_Validation(t *testing.T) {
	tests := []struct {
		name       string
		app        string
		entityID   string
		wantStatus int
	}{
		{"empty app returns 400", "", "bm-123", 400},
		{"empty entity_id returns 400", "karakeep", "", 400},
		{"both empty returns 400", "", "", 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New(fiber.Config{
				ErrorHandler: errorHandler,
			})
			app.Get("/:app/:entity_id/relations", getRelations)
			defer app.Shutdown()
			// Build query params; use "_" sentinel for empty values since
			// Fiber returns "" for both unset and empty query params.
			qApp := tt.app
			if qApp == "" {
				qApp = "_"
			}
			qEntity := tt.entityID
			if qEntity == "" {
				qEntity = "_"
			}
			url := "/x/id/relations?app=" + qApp + "&entity_id=" + qEntity
			req := httptest.NewRequest(fiber.MethodGet, url, nil)
			resp, _ := app.Test(req)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}
