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

	lifemod "github.com/flowline-io/flowbot/internal/modules/life"
	webmod "github.com/flowline-io/flowbot/internal/modules/web"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	pkgconfig "github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
)

type lifeWebAdapter struct {
	store.Adapter
	ent    *gen.Client
	uid    string
	scopes []string
}

func (a *lifeWebAdapter) Open(_ pkgconfig.StoreType) error { return nil }
func (a *lifeWebAdapter) Close() error                     { return nil }
func (a *lifeWebAdapter) IsOpen() bool                     { return true }
func (a *lifeWebAdapter) GetName() string                  { return "bdd-life-page" }
func (a *lifeWebAdapter) Stats() any                       { return nil }
func (a *lifeWebAdapter) GetDB() any                       { return a.ent }
func (a *lifeWebAdapter) GetClient() *gen.Client            { return a.ent }

func (a *lifeWebAdapter) ParameterGet(_ context.Context, flag string) (gen.Parameter, error) {
	return gen.Parameter{
		ID:        1,
		Flag:      flag,
		Params:    bddWebAuthParams(a.uid, a.scopes),
		ExpiredAt: time.Now().Add(time.Hour),
	}, nil
}

func (a *lifeWebAdapter) ParameterSet(ctx context.Context, flag string, params types.KV, expiredAt time.Time) error {
	return bddNoopParameterSet(ctx, flag, params, expiredAt)
}

var _ = Describe("Life Pages", Label("module", "web", "life"), func() {
	var (
		origDB      store.Adapter
		lifeAdapter *lifeWebAdapter
		testUID     string
	)

	BeforeEach(func() {
		origDB = store.Database
		testUID = "bdd-life-uid-" + types.Id()
		lifeAdapter = &lifeWebAdapter{
			ent:    EntClient,
			uid:    testUID,
			scopes: bddWebScopesUser(),
		}
		store.Database = lifeAdapter

		conf := json.RawMessage(`{"enabled":true,"auth":{"username":"admin","password":"flowbot-dev-pass"}}`)
		_ = webmod.InitForE2E(conf)
		webmod.MountForE2E(App)
		webmod.SetLifeService(lifemod.NewService(store.NewLifeStore(EntClient)))
		bddSeedAccessToken("life-token", lifeAdapter.uid, lifeAdapter.scopes)
	})

	AfterEach(func() {
		store.Database = origDB
		webmod.SetLifeService(nil)
	})

	Describe("auth gate", func() {
		It("redirects unauthenticated GET /life to login", func() {
			req := MakeRequest(http.MethodGet, "/service/web/life", nil)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusSeeOther))
			Expect(resp.Header.Get("Location")).To(ContainSubstring("/service/web/login"))
		})
	})

	Describe("authenticated pages", func() {
		It("renders dashboard", func() {
			req := MakeRequest(http.MethodGet, "/service/web/life", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "life-token"})
			webmod.AttachCSRFForTest(req)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			body := string(ReadBody(resp))
			Expect(body).To(ContainSubstring("Life"))
		})

		It("renders character page", func() {
			req := MakeRequest(http.MethodGet, "/service/web/life/character", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "life-token"})
			webmod.AttachCSRFForTest(req)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			body := string(ReadBody(resp))
			Expect(body).To(ContainSubstring("life-character"))
			Expect(body).To(ContainSubstring("life-class-form"))
			Expect(body).NotTo(ContainSubstring("life-goals-page"))
			Expect(body).NotTo(ContainSubstring("life-plan-page"))
		})

		It("renders goals page", func() {
			req := MakeRequest(http.MethodGet, "/service/web/life/goals", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "life-token"})
			webmod.AttachCSRFForTest(req)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			body := string(ReadBody(resp))
			Expect(body).To(ContainSubstring("life-goals-page"))
			Expect(body).To(ContainSubstring("life-goal-form"))
			Expect(body).To(ContainSubstring("life-goto-goals"))
		})

		It("renders plan page", func() {
			req := MakeRequest(http.MethodGet, "/service/web/life/plan", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "life-token"})
			webmod.AttachCSRFForTest(req)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			body := string(ReadBody(resp))
			Expect(body).To(ContainSubstring("life-plan-page"))
			Expect(body).To(ContainSubstring("life-plan-tree"))
			Expect(body).To(ContainSubstring("life-goto-plan"))
		})

		It("renders quests page", func() {
			req := MakeRequest(http.MethodGet, "/service/web/life/quests", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "life-token"})
			webmod.AttachCSRFForTest(req)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})

		It("renders inventory page", func() {
			req := MakeRequest(http.MethodGet, "/service/web/life/inventory", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "life-token"})
			webmod.AttachCSRFForTest(req)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			body := string(ReadBody(resp))
			Expect(body).To(ContainSubstring("life-equip-board"))
			Expect(body).To(ContainSubstring("life-equip-slot-armor"))
			Expect(body).To(ContainSubstring("Backpack"))
		})

		It("renders rewards page", func() {
			req := MakeRequest(http.MethodGet, "/service/web/life/rewards", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "life-token"})
			webmod.AttachCSRFForTest(req)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			body := string(ReadBody(resp))
			Expect(body).To(ContainSubstring("life-rewards-page"))
			Expect(body).To(ContainSubstring("life-rewards-create-form"))
			Expect(body).To(ContainSubstring("life-goto-rewards"))
		})

		It("renders stats shell and panel", func() {
			req := MakeRequest(http.MethodGet, "/service/web/life/stats", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "life-token"})
			webmod.AttachCSRFForTest(req)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			body := string(ReadBody(resp))
			Expect(body).To(ContainSubstring("life-stats-page"))
			Expect(body).To(ContainSubstring("life-stats-loader"))
			Expect(body).To(ContainSubstring("life-goto-stats"))
			Expect(body).To(ContainSubstring("life-stats.js"))

			panelReq := MakeRequest(http.MethodGet, "/service/web/life/stats/panel?tz=Asia/Shanghai", nil)
			panelReq.AddCookie(&http.Cookie{Name: "accessToken", Value: "life-token"})
			webmod.AttachCSRFForTest(panelReq)
			panelResp, err := App.Test(panelReq)
			Expect(err).NotTo(HaveOccurred())
			Expect(panelResp.StatusCode).To(Equal(http.StatusOK))
			panelBody := string(ReadBody(panelResp))
			Expect(panelBody).To(ContainSubstring("life-stats-container"))
			Expect(panelBody).To(ContainSubstring("life-stats-kpi"))
			Expect(panelBody).To(ContainSubstring("Asia/Shanghai"))
		})
	})
})
