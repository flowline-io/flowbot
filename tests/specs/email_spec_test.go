//go:build integration
// +build integration

package specs

import (
	"context"
	"net/http"

	hubmod "github.com/flowline-io/flowbot/internal/modules/hub"
	"github.com/flowline-io/flowbot/pkg/capability"
	capemail "github.com/flowline-io/flowbot/pkg/capability/email"
	"github.com/flowline-io/flowbot/pkg/hub"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Email Capability", Label("module", "email"), func() {
	BeforeEach(func() {
		hubmod.MountForE2E(App)
	})

	Describe("Webservice", func() {
		It("exposes health endpoint", func() {
			req := MakeRequest(http.MethodGet, "/service/email/health", nil)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Or(Equal(http.StatusOK), Equal(http.StatusBadRequest), Equal(http.StatusUnauthorized), Equal(http.StatusServiceUnavailable)))
		})

		It("lists messages via GET", func() {
			req := MakeRequest(http.MethodGet, "/service/email/messages?limit=5", nil)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Or(Equal(http.StatusOK), Equal(http.StatusBadRequest), Equal(http.StatusUnauthorized), Equal(http.StatusServiceUnavailable)))
		})

		It("searches messages via GET (read scope)", func() {
			req := MakeRequest(http.MethodGet, "/service/email/search?subject=test&limit=5", nil)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Or(Equal(http.StatusOK), Equal(http.StatusBadRequest), Equal(http.StatusUnauthorized), Equal(http.StatusServiceUnavailable)))
		})
	})

	Describe("Ability layer", func() {
		It("invokes health when configured", func() {
			result, err := capability.Invoke(context.Background(), hub.CapEmail, capemail.OpHealth, map[string]any{})
			if err != nil {
				Skip("email backend not configured: " + err.Error())
			}
			Expect(result).NotTo(BeNil())
		})

		It("rejects send without recipients", func() {
			_, err := capability.Invoke(context.Background(), hub.CapEmail, capemail.OpSend, map[string]any{
				"subject": "hi",
				"text":    "body",
			})
			if err == nil {
				Fail("expected validation error")
			}
		})
	})
})
