//go:build integration
// +build integration

package specs

import (
	"net/http"

	"github.com/bytedance/sonic"
	automatemod "github.com/flowline-io/flowbot/internal/modules/automate"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pipeline Module", Label("module", "pipeline"), func() {

	Describe("Webservice — apply / list / run", func() {
		BeforeEach(func() {
			automatemod.MountForE2E(App)
		})

		It("rejects apply without yaml", func() {
			body, _ := sonic.Marshal(map[string]any{})
			req := JSONRequest(http.MethodPost, "/service/automate/pipeline/apply", body)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Or(
				Equal(http.StatusBadRequest),
				Equal(http.StatusUnauthorized),
				Equal(http.StatusServiceUnavailable),
			))
		})

		It("rejects run without pipeline name", func() {
			body, _ := sonic.Marshal(map[string]any{"event": map[string]any{}})
			req := JSONRequest(http.MethodPost, "/service/automate/pipeline/run", body)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Or(
				Equal(http.StatusBadRequest),
				Equal(http.StatusUnauthorized),
				Equal(http.StatusServiceUnavailable),
			))
		})

		It("lists pipelines or reports unavailable", func() {
			req := JSONRequest(http.MethodGet, "/service/automate/pipeline/list", nil)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Or(
				Equal(http.StatusOK),
				Equal(http.StatusUnauthorized),
				Equal(http.StatusServiceUnavailable),
			))
		})

		It("rejects get without name", func() {
			req := JSONRequest(http.MethodGet, "/service/automate/pipeline/get/", nil)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Or(
				Equal(http.StatusBadRequest),
				Equal(http.StatusNotFound),
				Equal(http.StatusUnauthorized),
				Equal(http.StatusServiceUnavailable),
			))
		})

		It("rejects delete without name", func() {
			req := JSONRequest(http.MethodDelete, "/service/automate/pipeline/delete/", nil)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Or(
				Equal(http.StatusBadRequest),
				Equal(http.StatusNotFound),
				Equal(http.StatusUnauthorized),
				Equal(http.StatusServiceUnavailable),
			))
		})
	})
})
