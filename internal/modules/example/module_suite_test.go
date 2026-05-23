package example

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/flowline-io/flowbot/pkg/module"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/webservice"
)

func TestExampleModuleBDD(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Example Module BDD Suite")
}

var _ = ginkgo.Describe("Example Module", func() {
	var app *fiber.App

	ginkgo.BeforeEach(func() {
		app = fiber.New(fiber.Config{
			ErrorHandler: func(ctx fiber.Ctx, err error) error {
				code, _ := mapDomainErrorStatus(err)
				return ctx.Status(code).SendString(err.Error())
			},
		})
		allRules := append(webserviceRules, webhookRules...)
		module.Webservice(app, Name, webservice.Ruleset(allRules))
	})

	ginkgo.AfterEach(func() {
		_ = app.Shutdown()
	})

	ginkgo.It("returns existing example endpoint", func() {
		req := httptest.NewRequest("GET", "/service/example/example", nil)
		resp, err := app.Test(req)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(resp.StatusCode).To(gomega.Equal(200))
	})

	ginkgo.It("returns 400 for get without id", func() {
		req := httptest.NewRequest("GET", "/service/example/get", nil)
		resp, err := app.Test(req)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(resp.StatusCode).To(gomega.Equal(400))
	})

	ginkgo.It("returns 202 for webhook POST", func() {
		req := httptest.NewRequest("POST", "/service/example/webhook/example", nil)
		resp, err := app.Test(req)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(resp.StatusCode).To(gomega.Equal(202))
	})
})

// mapDomainErrorStatus maps a Flowbot error kind to an HTTP status code.
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
