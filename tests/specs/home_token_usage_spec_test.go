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

type homeWebAdapter struct {
	store.Adapter
	ent *gen.Client
	uid string
}

func (a *homeWebAdapter) Open(_ pkgconfig.StoreType) error { return nil }
func (a *homeWebAdapter) Close() error                     { return nil }
func (a *homeWebAdapter) IsOpen() bool                     { return true }
func (a *homeWebAdapter) GetName() string                  { return "bdd-home-token-usage" }
func (a *homeWebAdapter) Stats() any                       { return nil }
func (a *homeWebAdapter) GetDB() any                       { return a.ent }

func (a *homeWebAdapter) ParameterGet(_ context.Context, flag string) (gen.Parameter, error) {
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

var _ = Describe("Home Token Usage /home/token-usage", Label("home", "web"), func() {
	var (
		origDB  store.Adapter
		adapter *homeWebAdapter
	)

	BeforeEach(func() {
		origDB = store.Database
		adapter = &homeWebAdapter{
			ent: EntClient,
			uid: "bdd-home-user-" + types.Id(),
		}
		store.Database = adapter

		conf := json.RawMessage(`{"enabled":true,"auth":{"username":"admin","password":"flowbot-dev-pass"}}`)
		_ = webmod.InitForE2E(conf)
		webmod.MountForE2E(App)
	})

	AfterEach(func() {
		store.Database = origDB
	})

	Describe("GET /service/web/home/token-usage", func() {
		It("returns 303 when unauthenticated", func() {
			req := MakeRequest(http.MethodGet, "/service/web/home/token-usage?range=7d", nil)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusSeeOther))
		})

		It("returns 200 with token usage container when authenticated", func() {
			req := MakeRequest(http.MethodGet, "/service/web/home/token-usage?range=7d&groupBy=model", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "bdd-home-token"})
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			body := string(ReadBody(resp))
			Expect(body).To(ContainSubstring(`data-testid="token-usage-container"`))
			Expect(body).To(ContainSubstring("Token Usage"))
		})
	})
})
