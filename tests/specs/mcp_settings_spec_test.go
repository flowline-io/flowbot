//go:build integration

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

type mcpSettingsWebAdapter struct {
	store.Adapter
	ent *gen.Client
	uid string
}

func (a *mcpSettingsWebAdapter) Open(_ pkgconfig.StoreType) error { return nil }
func (a *mcpSettingsWebAdapter) Close() error                     { return nil }
func (a *mcpSettingsWebAdapter) IsOpen() bool                     { return true }
func (a *mcpSettingsWebAdapter) GetName() string                  { return "bdd-mcp-settings" }
func (a *mcpSettingsWebAdapter) Stats() any                       { return nil }
func (a *mcpSettingsWebAdapter) GetDB() any                       { return a.ent }
func (a *mcpSettingsWebAdapter) GetClient() *gen.Client           { return a.ent }

func (a *mcpSettingsWebAdapter) ParameterGet(_ context.Context, flag string) (gen.Parameter, error) {
	return gen.Parameter{
		ID:        1,
		Flag:      flag,
		Params:    bddWebAuthParams(a.uid, bddWebScopesAdmin()),
		ExpiredAt: time.Now().Add(time.Hour),
	}, nil
}

func (a *mcpSettingsWebAdapter) ParameterSet(ctx context.Context, flag string, params types.KV, expiredAt time.Time) error {
	return bddNoopParameterSet(ctx, flag, params, expiredAt)
}

func (a *mcpSettingsWebAdapter) Ping(_ context.Context) (time.Duration, error) {
	return time.Millisecond, nil
}

var _ = Describe("MCP settings /service/web/settings/mcp", Label("mcp", "web"), func() {
	var (
		origDB  store.Adapter
		adapter *mcpSettingsWebAdapter
		prevMCP pkgconfig.MCPConfig
	)

	BeforeEach(func() {
		origDB = store.Database
		adapter = &mcpSettingsWebAdapter{
			ent: EntClient,
			uid: "bdd-mcp-settings-user-" + types.Id(),
		}
		store.Database = adapter
		prevMCP = pkgconfig.App.MCP
		pkgconfig.App.MCP.Enabled = false

		conf := json.RawMessage(`{"enabled":true,"auth":{"username":"admin","password":"flowbot-dev-pass"}}`)
		_ = webmod.InitForE2E(conf)
		webmod.MountForE2E(App)
		bddSeedAccessToken("bdd-mcp-settings-token", adapter.uid, bddWebScopesAdmin())
	})

	AfterEach(func() {
		store.Database = origDB
		pkgconfig.App.MCP = prevMCP
	})

	It("redirects unauthenticated users to login", func() {
		req := MakeRequest(http.MethodGet, "/service/web/settings/mcp", nil)
		resp, err := App.Test(req)
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()
		Expect(resp.StatusCode).To(Equal(http.StatusSeeOther))
		Expect(resp.Header.Get("Location")).To(ContainSubstring("/service/web/login"))
	})

	It("shows the restart note while mcp.enabled is false", func() {
		req := MakeRequest(http.MethodGet, "/service/web/settings/mcp", nil)
		req.AddCookie(&http.Cookie{Name: "accessToken", Value: "bdd-mcp-settings-token"})
		webmod.AttachCSRFForTest(req)
		resp, err := App.Test(req)
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		body := string(ReadBody(resp))
		Expect(body).To(ContainSubstring(`data-testid="mcp-settings-page"`))
		Expect(body).To(ContainSubstring(`data-testid="mcp-restart-note"`))
		Expect(body).To(ContainSubstring(`data-testid="mcp-snippet-codex"`))
	})

	It("mints a token for the signed-in user", func() {
		req := MakeRequest(http.MethodPost, "/service/web/settings/mcp/token", []byte("preset=read&expires=2160h"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "accessToken", Value: "bdd-mcp-settings-token"})
		webmod.AttachCSRFForTest(req)
		resp, err := App.Test(req)
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		body := string(ReadBody(resp))
		Expect(body).To(ContainSubstring(`data-testid="mcp-token-alert"`))
		Expect(body).To(ContainSubstring(`data-testid="mcp-token"`))
	})
})
