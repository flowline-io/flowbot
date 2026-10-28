//go:build integration

package specs

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	automatemod "github.com/flowline-io/flowbot/internal/modules/automate"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/types"
)

var _ = Describe("Automate delete scope", Label("module", "automate", "auth"), func() {
	BeforeEach(func() {
		automatemod.MountForE2E(App)
	})

	It("rejects the MCP run preset token on automate delete", func() {
		token := "bdd-automate-run-" + types.Id()
		bddSeedAccessToken(token, "bdd-automate-run-user-"+types.Id(), auth.MCPRunScopes())

		for _, path := range []string{
			"/service/automate/pipeline/delete/bdd-none",
			"/service/automate/functions/delete/bdd-none",
			"/service/automate/workflow/delete/bdd-none",
		} {
			req := MakeRequest(http.MethodDelete, path, nil)
			req.Header.Set("X-AccessToken", token)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusForbidden), path)
			_ = resp.Body.Close()
		}
	})
})
