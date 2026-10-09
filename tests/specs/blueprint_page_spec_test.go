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
	"github.com/flowline-io/flowbot/pkg/pipeline"
	"github.com/flowline-io/flowbot/pkg/types"
)

type blueprintPageAdapter struct {
	store.Adapter
	ent    *gen.Client
	uid    string
	scopes []string
}

func (a *blueprintPageAdapter) Open(_ pkgconfig.StoreType) error { return nil }
func (a *blueprintPageAdapter) Close() error                     { return nil }
func (a *blueprintPageAdapter) IsOpen() bool                     { return true }
func (a *blueprintPageAdapter) GetName() string                  { return "bdd-blueprint-page" }
func (a *blueprintPageAdapter) Stats() any                       { return nil }
func (a *blueprintPageAdapter) GetDB() any                       { return a.ent }
func (a *blueprintPageAdapter) GetClient() *gen.Client           { return a.ent }

func (a *blueprintPageAdapter) ParameterGet(_ context.Context, flag string) (gen.Parameter, error) {
	return gen.Parameter{
		ID:        1,
		Flag:      flag,
		Params:    bddWebAuthParams(a.uid, a.scopes),
		ExpiredAt: time.Now().Add(time.Hour),
	}, nil
}

func (a *blueprintPageAdapter) ParameterSet(ctx context.Context, flag string, params types.KV, expiredAt time.Time) error {
	return bddNoopParameterSet(ctx, flag, params, expiredAt)
}

var _ = Describe("Blueprint library pages", Label("module", "web", "blueprint"), func() {
	var (
		origDB  store.Adapter
		adapter *blueprintPageAdapter
	)

	BeforeEach(func() {
		origDB = store.Database
		adapter = &blueprintPageAdapter{
			ent:    EntClient,
			uid:    "bdd-blueprint-uid-" + types.Id(),
			scopes: bddWebScopesAdmin(),
		}
		store.Database = adapter
		ps := store.NewPipelineStore(EntClient)
		catalog := store.PipelineCatalogAdapter{S: ps}
		pipeline.SetActiveBlueprintService(pipeline.NewBlueprintService(ps, catalog))
		pipeline.SetActiveService(pipeline.NewService(catalog))

		conf := json.RawMessage(`{"enabled":true,"auth":{"username":"admin","password":"flowbot-dev-pass"}}`)
		_ = webmod.InitForE2E(conf)
		webmod.MountForE2E(App)
		bddSeedAccessToken(adapter.uid, adapter.uid, adapter.scopes)
	})

	AfterEach(func() {
		pipeline.SetActiveBlueprintService(nil)
		pipeline.SetActiveService(nil)
		store.Database = origDB
	})

	Describe("GET /service/web/blueprints", func() {
		It("lists official templates", func() {
			req := MakeRequest(http.MethodGet, "/service/web/blueprints", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: adapter.uid})
			webmod.AttachCSRFForTest(req)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			body := string(ReadBody(resp))
			Expect(body).To(ContainSubstring(`data-testid="blueprint-table"`))
			Expect(body).To(ContainSubstring("webhook_notify"))
		})
	})

	Describe("GET /service/web/blueprints/:source/:id/download", func() {
		It("downloads an official blueprint YAML", func() {
			req := MakeRequest(http.MethodGet, "/service/web/blueprints/builtin/webhook_notify/download", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: adapter.uid})
			webmod.AttachCSRFForTest(req)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			body := string(ReadBody(resp))
			Expect(body).To(ContainSubstring("kind: pipeline_blueprint"))
			Expect(body).To(ContainSubstring("id: webhook_notify"))
		})
	})
})
