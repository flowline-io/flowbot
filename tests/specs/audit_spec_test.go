//go:build integration
// +build integration

package specs

import (
	"context"
	"net/http"
	"sync"

	"github.com/gofiber/fiber/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flowline-io/flowbot/internal/modules/lineage"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/types/audit"
)

var auditRoutesOnce sync.Once

func mountAuditRoutes(app *fiber.App) {
	auditRoutesOnce.Do(func() {
		lineage.MountForE2E(app)
	})
}

var _ = Describe("Audit list", Label("module", "audit"), func() {
	BeforeEach(func() {
		mountAuditRoutes(App)
	})

	It("returns 401 without a token", func() {
		req := JSONRequest(http.MethodGet, "/service/audit", nil)
		resp, err := App.Test(req)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
	})

	It("lists the caller uid and rejects another uid", func() {
		ctx := context.Background()
		auditor := store.NewAuditStore(EntClient)
		Expect(auditor.RecordSuccess(ctx, audit.Entry{
			Subject: &audit.Subject{SubjectType: "token", SubjectID: "audit-bdd-a", UID: "audit-bdd-a"},
			Action:  "mcp.tool.call",
			Target:  audit.Target{Type: "mcp_tool", ID: "audit-bdd-own"},
			Request: map[string]any{"tool": "audit-bdd-own"},
		})).To(Succeed())
		Expect(auditor.RecordSuccess(ctx, audit.Entry{
			Subject: &audit.Subject{SubjectType: "token", SubjectID: "audit-bdd-b", UID: "audit-bdd-b"},
			Action:  "mcp.tool.call",
			Target:  audit.Target{Type: "mcp_tool", ID: "audit-bdd-other"},
			Request: map[string]any{"tool": "audit-bdd-other"},
		})).To(Succeed())

		token, err := auth.NewToken()
		Expect(err).NotTo(HaveOccurred())
		bddSeedAccessToken(token, "audit-bdd-a", []string{auth.ScopeAuditRead})

		req := JSONRequest(http.MethodGet, "/service/audit?action=mcp.tool.call", nil)
		req.Header.Set("X-AccessToken", token)
		resp, err := App.Test(req)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		body := string(ReadBody(resp))
		Expect(body).To(ContainSubstring("audit-bdd-own"))
		Expect(body).NotTo(ContainSubstring("audit-bdd-other"))

		denied := JSONRequest(http.MethodGet, "/service/audit?subject_id=audit-bdd-b", nil)
		denied.Header.Set("X-AccessToken", token)
		deniedResp, err := App.Test(denied)
		Expect(err).NotTo(HaveOccurred())
		Expect(deniedResp.StatusCode).NotTo(Equal(http.StatusOK))
		Expect(string(ReadBody(deniedResp))).To(ContainSubstring("outside"))
	})
})
