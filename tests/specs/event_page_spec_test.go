//go:build integration
// +build integration

package specs

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	webmod "github.com/flowline-io/flowbot/internal/modules/web"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/dataevent"
	pkgconfig "github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
)

type eventWebAdapter struct {
	store.Adapter
	ent    *gen.Client
	uid    string
	scopes []string
}

func (a *eventWebAdapter) Open(_ pkgconfig.StoreType) error { return nil }
func (a *eventWebAdapter) Close() error                     { return nil }
func (a *eventWebAdapter) IsOpen() bool                     { return true }
func (a *eventWebAdapter) GetName() string                  { return "bdd-event-page" }
func (a *eventWebAdapter) Stats() any                       { return nil }
func (a *eventWebAdapter) GetDB() any                       { return a.ent }

func (a *eventWebAdapter) ParameterGet(_ context.Context, flag string) (gen.Parameter, error) {
	return gen.Parameter{
		ID:    1,
		Flag:  flag,
		Params: map[string]any{
			"uid":    a.uid,
			"topic":  "test",
			"scopes": a.scopes,
		},
		ExpiredAt: time.Now().Add(time.Hour),
	}, nil
}

var _ = Describe("Events Pages", Label("module", "web"), func() {
	var (
		origDB       store.Adapter
		adminAdapter *eventWebAdapter
		userAdapter  *eventWebAdapter
		seedEvents   []*gen.DataEvent
	)

	BeforeEach(func() {
		origDB = store.Database
		adminAdapter = &eventWebAdapter{
			ent:    EntClient,
			uid:    "bdd-admin-uid-" + types.Id(),
			scopes: []string{"admin:*", "read", "write"},
		}
		userAdapter = &eventWebAdapter{
			ent:    EntClient,
			uid:    "bdd-user-uid-" + types.Id(),
			scopes: []string{"read", "write"},
		}
		store.Database = adminAdapter

		conf := json.RawMessage(`{"enabled":true,"auth":{"username":"admin","password":"flowbot-dev-pass"}}`)
		_ = webmod.InitForE2E(conf)
		webmod.MountForE2E(App)

		// Seed regular event
		e1 := EntClient.DataEvent.Create().
			SetEventID("bdd-event-regular-" + types.Id()).
			SetEventType("bookmark.created").
			SetSource("test-agent").
			SetData(map[string]any{"url": "https://example.com", "title": "BDD Event"}).
			SetCreatedAt(time.Now().Add(-10 * time.Minute)).
			SaveX(context.Background())

		// Seed webhook event 1
		e2 := EntClient.DataEvent.Create().
			SetEventID("bdd-event-webhook-1-" + types.Id()).
			SetEventType("webhook.push").
			SetSource("github").
			SetData(map[string]any{
				"_webhook_method":  "POST",
				"_webhook_headers": map[string]any{"Content-Type": "application/json"},
				"_webhook_body":    `{"ref":"refs/heads/main"}`,
			}).
			SetCreatedAt(time.Now().Add(-5 * time.Minute)).
			SaveX(context.Background())

		// Seed webhook event 2
		e3 := EntClient.DataEvent.Create().
			SetEventID("bdd-event-webhook-2-" + types.Id()).
			SetEventType("webhook.issue").
			SetSource("github").
			SetData(map[string]any{
				"_webhook_method": "POST",
				"_webhook_headers": map[string]any{"X-Hub-Signature": "sha1=abc"},
				"_webhook_body":    `{"action":"opened"}`,
			}).
			SetCreatedAt(time.Now()).
			SaveX(context.Background())

		seedEvents = []*gen.DataEvent{e1, e2, e3}

		types.EventFilterCache.Hydrate(
			[]string{"test-agent", "github"},
			[]string{"bookmark.created", "webhook.push", "webhook.issue"},
		)
	})

	AfterEach(func() {
		for _, e := range seedEvents {
			EntClient.DataEvent.Delete().Where(
				dataevent.ID(e.ID),
			).ExecX(context.Background())
		}
		store.Database = origDB
	})

	Describe("GET /events", func() {
		Context("with admin scope", func() {
			It("returns the events page with tabs and data", func() {
				req := MakeRequest(http.MethodGet, "/service/web/events", nil)
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: adminAdapter.uid})
				webmod.AttachCSRFForTest(req)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				body := string(ReadBody(resp))
				Expect(body).To(ContainSubstring("data-events"))
				Expect(body).To(ContainSubstring("test-agent"))
				Expect(body).To(ContainSubstring("github"))
			})
		})

		Context("with non-admin scope", func() {
			It("returns 403 forbidden", func() {
				store.Database = userAdapter

				req := MakeRequest(http.MethodGet, "/service/web/events", nil)
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: userAdapter.uid})
				webmod.AttachCSRFForTest(req)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
				Expect(string(ReadBody(resp))).To(Equal("Admin access required"))
			})
		})

		Context("without authentication", func() {
			It("redirects to login page", func() {
				req := MakeRequest(http.MethodGet, "/service/web/events", nil)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusSeeOther))
				Expect(resp.Header.Get("Location")).To(ContainSubstring("/service/web/login"))
			})
		})
	})

	Describe("GET /events/data-events", func() {
		Context("with admin scope", func() {
			It("returns the data events table fragment", func() {
				req := MakeRequest(http.MethodGet, "/service/web/events/data-events", nil)
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: adminAdapter.uid})
				webmod.AttachCSRFForTest(req)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				body := string(ReadBody(resp))
				Expect(body).To(ContainSubstring("bookmark.created"))
				Expect(body).To(ContainSubstring("webhook.push"))
				Expect(body).To(ContainSubstring("webhook.issue"))
			})

			It("filters by source query parameter", func() {
				req := MakeRequest(http.MethodGet, "/service/web/events/data-events?source=test-agent", nil)
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: adminAdapter.uid})
				webmod.AttachCSRFForTest(req)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				body := string(ReadBody(resp))
				Expect(body).To(ContainSubstring("bookmark.created"))
				Expect(body).NotTo(ContainSubstring("webhook.push"))
			})
		})
	})

	Describe("GET /events/webhook-logs", func() {
		Context("with admin scope", func() {
			It("returns the webhook logs table fragment", func() {
				req := MakeRequest(http.MethodGet, "/service/web/events/webhook-logs", nil)
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: adminAdapter.uid})
				webmod.AttachCSRFForTest(req)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				body := string(ReadBody(resp))
				Expect(body).To(ContainSubstring("webhook.push"))
				Expect(body).To(ContainSubstring("webhook.issue"))
				Expect(body).NotTo(ContainSubstring("bookmark.created"))
			})
		})
	})

	Describe("GET /events/payload/:eventID", func() {
		Context("with admin scope", func() {
			It("returns payload detail for a regular event", func() {
				req := MakeRequest(http.MethodGet, "/service/web/events/payload/"+seedEvents[0].EventID, nil)
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: adminAdapter.uid})
				webmod.AttachCSRFForTest(req)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				body := string(ReadBody(resp))
				Expect(body).To(ContainSubstring("https://example.com"))
			})

			It("returns payload detail for a webhook event", func() {
				req := MakeRequest(http.MethodGet, "/service/web/events/payload/"+seedEvents[1].EventID, nil)
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: adminAdapter.uid})
				webmod.AttachCSRFForTest(req)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				body := string(ReadBody(resp))
				Expect(body).To(ContainSubstring("Content-Type"))
				Expect(body).To(ContainSubstring("refs/heads/main"))
			})

			It("returns not-found for a non-existent eventID", func() {
				req := MakeRequest(http.MethodGet, "/service/web/events/payload/bdd-no-such-event", nil)
				req.AddCookie(&http.Cookie{Name: "accessToken", Value: adminAdapter.uid})
				webmod.AttachCSRFForTest(req)
				resp, err := App.Test(req)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(string(ReadBody(resp))).To(ContainSubstring("Event not found"))
			})
		})
	})
})
