//go:build integration
// +build integration

package specs

import (
	"encoding/json"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	webmod "github.com/flowline-io/flowbot/internal/modules/web"
	"github.com/flowline-io/flowbot/internal/store"
)

var _ = Describe("Health Dashboard /healthz", Label("health", "web"), func() {
	var origDB store.Adapter

	BeforeEach(func() {
		origDB = store.Database
		conf := json.RawMessage(`{"enabled":true,"auth":{"username":"admin","password":"admin"}}`)
		_ = webmod.InitForE2E(conf)
		webmod.MountForE2E(App)
	})

	AfterEach(func() {
		store.Database = origDB
	})

	Describe("GET /service/web/healthz", func() {
		It("returns 200 and renders all four metric sections", func() {
			req := MakeRequest(http.MethodGet, "/service/web/healthz", nil)
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
