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
	"github.com/flowline-io/flowbot/pkg/homelab"
)

// homelabWebAdapter satisfies store.Adapter for BDD homelab registry page tests,
// delegating GetDB to the shared EntClient and providing stub auth.
type homelabWebAdapter struct {
	store.Adapter
	ent *gen.Client
}

func (a *homelabWebAdapter) Open(_ pkgconfig.StoreType) error { return nil }
func (a *homelabWebAdapter) Close() error                     { return nil }
func (a *homelabWebAdapter) IsOpen() bool                     { return true }
func (a *homelabWebAdapter) GetName() string                  { return "bdd-homelab-registry" }
func (a *homelabWebAdapter) Stats() any                       { return nil }
func (a *homelabWebAdapter) GetDB() any                       { return a.ent }

func (a *homelabWebAdapter) ParameterGet(_ context.Context, _ string) (gen.Parameter, error) {
	return gen.Parameter{
		ID:   1,
		Flag: "bdd-homelab",
		Params: map[string]any{
			"uid":    "bdd-homelab-uid",
			"topic":  "test",
			"scopes": []string{"admin:*"},
		},
		ExpiredAt: time.Now().Add(time.Hour),
	}, nil
}

var _ = Describe("Homelab Registry UI", Label("module", "web"), func() {
	var origDB store.Adapter

	BeforeEach(func() {
		origDB = store.Database
		store.Database = &homelabWebAdapter{ent: EntClient}

		conf := json.RawMessage(`{"enabled":true,"auth":{"username":"admin","password":"flowbot-dev-pass"}}`)
		_ = webmod.InitForE2E(conf)
		webmod.MountForE2E(App)

		homelab.SetRunRescan(func() error { return nil })

		homelab.DefaultRegistry.Replace([]homelab.App{
			{
				Name:   "gitea",
				Path:   "/apps/gitea",
				Status: homelab.AppStatusRunning,
				Health: homelab.HealthHealthy,
				Services: []homelab.ComposeService{
					{
						Name:      "gitea",
						Image:     "gitea/gitea:1.22.3",
						Container: "gitea-app",
						Ports: []homelab.PortMapping{
							{HostPort: "3000", Container: "3000", Protocol: "tcp"},
							{HostPort: "2222", Container: "22", Protocol: "tcp"},
						},
					},
				},
				Capabilities: []homelab.AppCapability{
					{
						Capability: homelab.CapGitea,
						Endpoint: &homelab.EndpointInfo{
							BaseURL: "http://gitea.local:3000",
							Health:  "/api/health",
						},
					},
				},
			},
			{
				Name:   "karakeep",
				Path:   "/apps/karakeep",
				Status: homelab.AppStatusStopped,
				Health: homelab.HealthUnknown,
				Capabilities: []homelab.AppCapability{
					{Capability: homelab.CapKarakeep},
				},
			},
		})
	})

	AfterEach(func() {
		store.Database = origDB
		homelab.DefaultRegistry.Replace(nil)
		homelab.SetRunRescan(nil)
	})

	Describe("GET /service/web/homelab", func() {
		It("renders the registry page with app cards", func() {
			req := MakeRequest(http.MethodGet, "/service/web/homelab", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "bdd-homelab-token"})
			webmod.AttachCSRFForTest(req)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			body := ReadBody(resp)
			Expect(string(body)).To(ContainSubstring("Homelab Registry"))
			Expect(string(body)).To(ContainSubstring("gitea"))
			Expect(string(body)).To(ContainSubstring("karakeep"))
		})

		It("shows empty state when no apps are registered", func() {
			homelab.DefaultRegistry.Replace(nil)

			req := MakeRequest(http.MethodGet, "/service/web/homelab", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "bdd-homelab-token"})
			webmod.AttachCSRFForTest(req)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			body := ReadBody(resp)
			Expect(string(body)).To(ContainSubstring("Homelab Registry"))
			Expect(string(body)).To(ContainSubstring("No apps discovered"))
		})
	})

	Describe("GET /service/web/homelab/:name", func() {
		It("renders detail page with services and endpoints", func() {
			req := MakeRequest(http.MethodGet, "/service/web/homelab/gitea", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "bdd-homelab-token"})
			webmod.AttachCSRFForTest(req)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			body := ReadBody(resp)
			Expect(string(body)).To(ContainSubstring("gitea"))
			Expect(string(body)).To(ContainSubstring("1.22.3"))
			Expect(string(body)).To(ContainSubstring("gitea/gitea"))
			Expect(string(body)).To(ContainSubstring("http://gitea.local:3000"))
		})

		It("returns 404 for unknown app", func() {
			req := MakeRequest(http.MethodGet, "/service/web/homelab/nonexistent", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "bdd-homelab-token"})
			webmod.AttachCSRFForTest(req)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
		})
	})

	Describe("POST /service/web/homelab/rescan", func() {
		It("returns HX-Redirect header", func() {
			req := MakeRequest(http.MethodPost, "/service/web/homelab/rescan", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "bdd-homelab-token"})
			webmod.AttachCSRFForTest(req)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(resp.Header.Get("HX-Redirect")).To(Equal("/service/web/homelab"))
		})
	})
})
