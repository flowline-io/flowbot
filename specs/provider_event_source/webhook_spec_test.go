package provider_event_source_test

import (
	"errors"
	"net/http/httptest"
	"strings"

	"github.com/gofiber/fiber/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/types"
)

type stubBDDConverter struct {
	path         string
	shouldFail   bool
	convertError bool
}

func (s *stubBDDConverter) WebhookPath() string { return s.path }
func (s *stubBDDConverter) VerifySignature(_ map[string]string, _ []byte) error {
	if s.shouldFail {
		return errors.New("bad signature")
	}
	return nil
}
func (s *stubBDDConverter) Convert(_ []byte, _ map[string]string) ([]types.DataEvent, error) {
	if s.convertError {
		return nil, errors.New("bad payload")
	}
	return nil, nil
}

var _ = Describe("Inbound Webhook", func() {
	It("returns 202 for valid webhook", func() {
		app := fiber.New()
		mgr := ability.NewEventSourceManager(nil, nil, nil)
		mgr.RegisterWebhook(&stubBDDConverter{
			path: "github/events",
		})
		app.Post("/webhook/provider/*", mgr.WebhookHandler())

		req := httptest.NewRequest("POST", "/webhook/provider/github/events",
			strings.NewReader(`{"action":"created"}`))
		resp, _ := app.Test(req)
		Expect(resp.StatusCode).To(Equal(fiber.StatusAccepted))
	})

	It("returns 404 for unknown webhook path", func() {
		app := fiber.New()
		mgr := ability.NewEventSourceManager(nil, nil, nil)
		app.Post("/webhook/provider/*", mgr.WebhookHandler())

		req := httptest.NewRequest("POST", "/webhook/provider/unknown/hooks", nil)
		resp, _ := app.Test(req)
		Expect(resp.StatusCode).To(Equal(fiber.StatusNotFound))
	})

	It("returns 401 for failed signature verification", func() {
		app := fiber.New()
		mgr := ability.NewEventSourceManager(nil, nil, nil)
		mgr.RegisterWebhook(&stubBDDConverter{
			path:       "secure/service",
			shouldFail: true,
		})
		app.Post("/webhook/provider/*", mgr.WebhookHandler())

		req := httptest.NewRequest("POST", "/webhook/provider/secure/service",
			strings.NewReader(`{}`))
		resp, _ := app.Test(req)
		Expect(resp.StatusCode).To(Equal(fiber.StatusUnauthorized))
	})

	It("returns 400 for convert error", func() {
		app := fiber.New()
		mgr := ability.NewEventSourceManager(nil, nil, nil)
		mgr.RegisterWebhook(&stubBDDConverter{
			path:         "bad/payload",
			convertError: true,
		})
		app.Post("/webhook/provider/*", mgr.WebhookHandler())

		req := httptest.NewRequest("POST", "/webhook/provider/bad/payload",
			strings.NewReader(`invalid`))
		resp, _ := app.Test(req)
		Expect(resp.StatusCode).To(Equal(fiber.StatusBadRequest))
	})

	It("returns 404 for empty path", func() {
		app := fiber.New()
		mgr := ability.NewEventSourceManager(nil, nil, nil)
		app.Post("/webhook/provider/*", mgr.WebhookHandler())

		req := httptest.NewRequest("POST", "/webhook/provider/", nil)
		resp, _ := app.Test(req)
		Expect(resp.StatusCode).To(Equal(fiber.StatusNotFound))
	})
})
