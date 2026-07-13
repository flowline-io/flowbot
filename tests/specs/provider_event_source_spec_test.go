//go:build integration
// +build integration

package specs

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/types"
)

var _ = Describe("Provider Event Source", Label("event_source"), func() {
	Describe("Inbound Webhook", func() {
		It("returns 202 for valid webhook", func() {
			app := fiber.New()
			mgr := capability.NewEventSourceManager(nil, nil, nil)
			mgr.RegisterWebhook(&stubConverter{path: "github/events"})
			app.Post("/webhook/provider/*", mgr.WebhookHandler())

			req := httptest.NewRequest("POST", "/webhook/provider/github/events",
				strings.NewReader(`{"action":"created"}`))
			resp, _ := app.Test(req)
			Expect(resp.StatusCode).To(Equal(fiber.StatusAccepted))
		})

		It("returns 404 for unknown webhook path", func() {
			app := fiber.New()
			mgr := capability.NewEventSourceManager(nil, nil, nil)
			app.Post("/webhook/provider/*", mgr.WebhookHandler())

			req := httptest.NewRequest("POST", "/webhook/provider/unknown/hooks", http.NoBody)
			resp, _ := app.Test(req)
			Expect(resp.StatusCode).To(Equal(fiber.StatusNotFound))
		})

		It("returns 401 for failed signature verification", func() {
			app := fiber.New()
			mgr := capability.NewEventSourceManager(nil, nil, nil)
			mgr.RegisterWebhook(&stubConverter{
				path:    "secure/service",
				sigFail: true,
			})
			app.Post("/webhook/provider/*", mgr.WebhookHandler())

			req := httptest.NewRequest("POST", "/webhook/provider/secure/service",
				strings.NewReader(`{}`))
			resp, _ := app.Test(req)
			Expect(resp.StatusCode).To(Equal(fiber.StatusUnauthorized))
		})

		It("returns 400 for convert error", func() {
			app := fiber.New()
			mgr := capability.NewEventSourceManager(nil, nil, nil)
			mgr.RegisterWebhook(&stubConverter{
				path:       "bad/payload",
				convertErr: true,
			})
			app.Post("/webhook/provider/*", mgr.WebhookHandler())

			req := httptest.NewRequest("POST", "/webhook/provider/bad/payload",
				strings.NewReader(`invalid`))
			resp, _ := app.Test(req)
			Expect(resp.StatusCode).To(Equal(fiber.StatusBadRequest))
		})
	})

	Describe("Cron Polling", func() {
		It("registers polling resource and starts without error", func() {
			mgr := capability.NewEventSourceManager(nil, nil, nil)
			r := &stubPollRes{name: "test/bookmarks"}
			mgr.RegisterPolling(r)
			Expect(mgr.Start(context.Background())).To(Succeed())
			Expect(mgr.Stop(context.Background())).To(Succeed())
		})

		It("starts with empty pollers without error", func() {
			mgr := capability.NewEventSourceManager(nil, nil, nil)
			Expect(mgr.Start(context.Background())).To(Succeed())
			Expect(mgr.Stop(context.Background())).To(Succeed())
		})
	})

	Describe("Polling State", func() {
		It("persists and recovers cursor state", func() {
			state := capability.NewPollingState(nil)
			state.Update("test/recovery", capability.PollingEntry{
				Cursor:      "cursor-42",
				KnownHashes: map[string]string{"k1": "h1"},
			})
			state.MarkDirty("test/recovery")
			Expect(state.Flush(context.Background())).To(Succeed())

			entry := state.Get("test/recovery")
			Expect(entry.Cursor).To(Equal("cursor-42"))
			Expect(entry.KnownHashes).To(HaveKeyWithValue("k1", "h1"))
		})
	})
})

type stubConverter struct {
	path       string
	sigFail    bool
	convertErr bool
}

func (s *stubConverter) WebhookPath() string { return s.path }
func (s *stubConverter) VerifySignature(_ map[string]string, _ []byte) error {
	if s.sigFail {
		return errors.New("bad signature")
	}
	return nil
}
func (s *stubConverter) Convert(_ []byte, _ map[string]string) ([]types.DataEvent, error) {
	if s.convertErr {
		return nil, errors.New("bad payload")
	}
	return nil, nil
}

type stubPollRes struct {
	name string
}

func (r *stubPollRes) ResourceName() string           { return r.name }
func (r *stubPollRes) DefaultInterval() time.Duration { return time.Hour }
func (r *stubPollRes) DiffKey(item any) string        { return "" }
func (r *stubPollRes) ContentHash(item any) string    { return "" }
func (r *stubPollRes) CursorField() string            { return "id" }
func (r *stubPollRes) List(ctx context.Context, cursor string) (capability.PollResult, error) {
	return capability.PollResult{}, nil
}
