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
	pkgconfig "github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
)

type healthzWebAdapter struct {
	store.Adapter
	ent *gen.Client
	uid string
}

func (a *healthzWebAdapter) Open(_ pkgconfig.StoreType) error { return nil }
func (a *healthzWebAdapter) Close() error                     { return nil }
func (a *healthzWebAdapter) IsOpen() bool                     { return true }
func (a *healthzWebAdapter) GetName() string                  { return "bdd-healthz" }
func (a *healthzWebAdapter) Stats() any                       { return nil }
func (a *healthzWebAdapter) GetDB() any                       { return a.ent }

func (a *healthzWebAdapter) ParameterGet(_ context.Context, flag string) (gen.Parameter, error) {
	return gen.Parameter{
		ID:   1,
		Flag: flag,
		Params: map[string]any{
			"uid":    a.uid,
			"topic":  "test",
			"scopes": []string{"admin:*"},
		},
		ExpiredAt: time.Now().Add(time.Hour),
	}, nil
}

func (a *healthzWebAdapter) Ping(_ context.Context) (time.Duration, error) {
	return time.Millisecond, nil
}

var _ = Describe("Health Dashboard /healthz", Label("health", "web"), func() {
	var (
		origDB  store.Adapter
		adapter *healthzWebAdapter
	)

	BeforeEach(func() {
		origDB = store.Database
		adapter = &healthzWebAdapter{
			ent: EntClient,
			uid: "bdd-healthz-user-" + types.Id(),
		}
		store.Database = adapter

		conf := json.RawMessage(`{"enabled":true,"auth":{"username":"admin","password":"flowbot-dev-pass"}}`)
		_ = webmod.InitForE2E(conf)
		webmod.MountForE2E(App)
	})

	AfterEach(func() {
		store.Database = origDB
	})

	Describe("GET /service/web/healthz", func() {
		It("redirects unauthenticated users to login", func() {
			req := MakeRequest(http.MethodGet, "/service/web/healthz", nil)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusSeeOther))
			Expect(resp.Header.Get("Location")).To(ContainSubstring("/service/web/login"))
		})

		It("returns 200 and renders all four metric sections when authenticated", func() {
			req := MakeRequest(http.MethodGet, "/service/web/healthz", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "bdd-healthz-token"})
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			body := string(ReadBody(resp))
			Expect(body).To(ContainSubstring("System Health"))
			Expect(body).To(ContainSubstring("Database Latency"))
			Expect(body).To(ContainSubstring("PostgreSQL"))
			Expect(body).To(ContainSubstring("Redis"))
			Expect(body).To(ContainSubstring("Runtime"))
			Expect(body).To(ContainSubstring("Goroutines"))
			Expect(body).To(ContainSubstring("Heap Alloc"))
			Expect(body).To(ContainSubstring("Capability Status"))
			Expect(body).To(ContainSubstring("Recent Errors"))
		})

		It("has HTMX auto-refresh attributes on the status section", func() {
			req := MakeRequest(http.MethodGet, "/service/web/healthz", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "bdd-healthz-token"})
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			body := string(ReadBody(resp))
			Expect(body).To(ContainSubstring(`hx-trigger="every 30s"`))
			Expect(body).To(ContainSubstring(`hx-get="/service/web/healthz"`))
		})

		It("renders partial status when requested via HTMX", func() {
			req := MakeRequest(http.MethodGet, "/service/web/healthz", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "bdd-healthz-token"})
			req.Header.Set("HX-Request", "true")
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			body := string(ReadBody(resp))
			Expect(body).NotTo(ContainSubstring("System Health"))
			Expect(body).To(ContainSubstring("Database Latency"))
			Expect(body).To(ContainSubstring(`hx-trigger="every 30s"`))
		})
	})
})
