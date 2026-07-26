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

func (a *lifeWebAdapter) ParameterGet(_ context.Context, flag string) (gen.Parameter, error) {
	return gen.Parameter{
		ID:   1,
		Flag: flag,
		Params: map[string]any{
			"uid":    a.uid,
			"topic":  "test",
			"scopes": a.scopes,
		},
		ExpiredAt: time.Now().Add(time.Hour),
	}, nil
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
			scopes: []string{"read", "write"},
		}
		store.Database = lifeAdapter

		conf := json.RawMessage(`{"enabled":true,"auth":{"username":"admin","password":"flowbot-dev-pass"}}`)
		_ = webmod.InitForE2E(conf)
		webmod.MountForE2E(App)
		webmod.SetLifeService(lifemod.NewService(store.NewLifeStore(EntClient)))
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
		})
	})
})
