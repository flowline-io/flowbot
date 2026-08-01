//go:build integration
// +build integration

package specs

import (
	"net/http"

	"github.com/bytedance/sonic"
	pipelinemod "github.com/flowline-io/flowbot/internal/modules/pipeline"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pipeline Module", Label("module", "pipeline"), func() {

	Describe("Webservice — apply / list / run", func() {
		BeforeEach(func() {
			pipelinemod.MountForE2E(App)
		})

		It("rejects apply without yaml", func() {
			body, _ := sonic.Marshal(map[string]any{})
			req := JSONRequest(http.MethodPost, "/service/pipeline/apply", body)
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
			req := JSONRequest(http.MethodPost, "/service/pipeline/run", body)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Or(
				Equal(http.StatusBadRequest),
				Equal(http.StatusUnauthorized),
				Equal(http.StatusServiceUnavailable),
			))
		})

		It("lists pipelines or reports unavailable", func() {
			req := JSONRequest(http.MethodGet, "/service/pipeline/list", nil)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Or(
				Equal(http.StatusOK),
				Equal(http.StatusUnauthorized),
				Equal(http.StatusServiceUnavailable),
			))
		})

		It("rejects get without name", func() {
			req := JSONRequest(http.MethodGet, "/service/pipeline/get/", nil)
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
			req := JSONRequest(http.MethodDelete, "/service/pipeline/delete/", nil)
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
