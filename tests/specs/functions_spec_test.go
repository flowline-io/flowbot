//go:build integration
// +build integration

package specs

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	functionsmod "github.com/flowline-io/flowbot/internal/modules/functions"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/agent/dcg"
	"github.com/flowline-io/flowbot/pkg/agent/env"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/capability"
	capfunctions "github.com/flowline-io/flowbot/pkg/capability/functions"
	pkgexec "github.com/flowline-io/flowbot/pkg/exec"
	pkgfunctions "github.com/flowline-io/flowbot/pkg/functions"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

type osFunctionExecProvider struct{}

func (osFunctionExecProvider) ExecConfig(_ context.Context) (pkgexec.Config, error) {
	return pkgexec.Config{
		Env:       env.Default(),
		Timeout:   30 * time.Second,
		MaxOutput: pkgfunctions.MaxJSONBytes,
	}, nil
}

func functionsUserToken() string {
	token, err := auth.NewToken()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	bddSeedAccessToken(token, "functions-bdd-user", []string{auth.ScopeFunctionRead, auth.ScopeFunctionRun})
	return token
}

func functionsAuthRequest(method, path, accessToken string, body []byte) *http.Request {
	req := JSONRequest(method, path, body)
	if accessToken != "" {
		req.Header.Set("X-AccessToken", accessToken)
	}
	return req
}

var _ = Describe("Named Functions Module", Label("module", "functions"), func() {
	var prev *pkgfunctions.Service

	BeforeEach(func() {
		functionsmod.MountForE2E(App)
		gomega.Expect(functionsmod.InitForE2E(nil)).To(gomega.Succeed())

		prev = pkgfunctions.ActiveService()
		catalog := store.NewFunctionCatalogAdapter(store.NewFunctionStore(EntClient))
		svc := pkgfunctions.NewService(catalog, osFunctionExecProvider{})
		svc.SetChecker(dcg.AllowAllChecker{})
		pkgfunctions.SetActiveService(svc)
	})

	AfterEach(func() {
		pkgfunctions.SetActiveService(prev)
	})

	It("rejects apply without function scopes", func() {
		body, _ := sonic.Marshal(map[string]any{
			"metadata":   "name: x\nhttp:\n  auth:\n    token: t\n",
			"entrypoint": "main.py",
			"source":     "print(1)",
		})
		req := functionsAuthRequest(http.MethodPost, "/service/functions/apply", "", body)
		resp, err := App.Test(req)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(resp.StatusCode).To(gomega.Equal(http.StatusUnauthorized))
	})

	It("applies a function directory bundle and rejects unauthenticated call", func() {
		token := functionsUserToken()
		name := "echo-fn-" + types.Id()
		meta := "name: " + name + "\nhttp:\n  auth:\n    token: secret-token\nenv:\n  mode: test\n"
		source := "import json,sys\nmsg=json.load(sys.stdin)\nprint(json.dumps({\"ok\": True, \"mode\": msg[\"env\"][\"mode\"]}))\n"
		body, _ := sonic.Marshal(map[string]any{
			"metadata":   meta,
			"entrypoint": "main.py",
			"source":     source,
		})
		applyReq := functionsAuthRequest(http.MethodPost, "/service/functions/apply", token, body)
		applyResp, err := App.Test(applyReq)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(applyResp.StatusCode).To(gomega.Equal(http.StatusOK))
		var applyOut protocol.Response
		gomega.Expect(sonic.Unmarshal(ReadBody(applyResp), &applyOut)).To(gomega.Succeed())
		gomega.Expect(applyOut.Status).To(gomega.Equal(protocol.Success))

		eventBody, _ := sonic.Marshal(map[string]any{"n": 1})
		unauthReq := JSONRequest(http.MethodPost, "/service/functions/call/"+name, eventBody)
		unauthResp, err := App.Test(unauthReq)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(unauthResp.StatusCode).To(gomega.Or(
			gomega.Equal(http.StatusUnauthorized),
			gomega.Equal(http.StatusBadRequest),
		))
	})

	It("invokes a published function with X-Webhook-Token and returns JSON", func() {
		token := functionsUserToken()
		name := "echo-fn-" + types.Id()
		meta := "name: " + name + "\nhttp:\n  auth:\n    token: secret-token\nenv:\n  mode: test\n"
		source := "import json,sys\nmsg=json.load(sys.stdin)\nprint(json.dumps({\"ok\": True, \"mode\": msg[\"env\"][\"mode\"], \"n\": msg[\"event\"][\"n\"]}))\n"
		body, _ := sonic.Marshal(map[string]any{
			"metadata":   meta,
			"entrypoint": "main.py",
			"source":     source,
		})
		applyReq := functionsAuthRequest(http.MethodPost, "/service/functions/apply", token, body)
		applyResp, err := App.Test(applyReq)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(applyResp.StatusCode).To(gomega.Equal(http.StatusOK))

		eventBody, _ := sonic.Marshal(map[string]any{"n": 7})
		callReq := JSONRequest(http.MethodPost, "/service/functions/call/"+name, eventBody)
		callReq.Header.Set("X-Webhook-Token", "secret-token")
		callResp, err := App.Test(callReq, fiber.TestConfig{Timeout: 45 * time.Second})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(callResp.StatusCode).To(gomega.Equal(http.StatusOK))
		var callOut protocol.Response
		gomega.Expect(sonic.Unmarshal(ReadBody(callResp), &callOut)).To(gomega.Succeed())
		gomega.Expect(callOut.Status).To(gomega.Equal(protocol.Success))
		data, ok := callOut.Data.(map[string]any)
		gomega.Expect(ok).To(gomega.BeTrue())
		gomega.Expect(data["status"]).To(gomega.Equal("succeeded"))
		result, ok := data["result"].(map[string]any)
		gomega.Expect(ok).To(gomega.BeTrue())
		gomega.Expect(result["ok"]).To(gomega.Equal(true))
		gomega.Expect(result["mode"]).To(gomega.Equal("test"))
	})

	It("applies via directory artifact and invokes through capability without HTTP secrets", func() {
		token := functionsUserToken()
		name := "dir-fn-" + types.Id()
		dir := GinkgoT().TempDir()
		meta := "name: " + name + "\nhttp:\n  auth:\n    token: secret-token\nenv:\n  mode: dir\n"
		source := "import json,sys\nmsg=json.load(sys.stdin)\nprint(json.dumps({\"ok\": True, \"mode\": msg[\"env\"][\"mode\"]}))\n"
		gomega.Expect(os.WriteFile(filepath.Join(dir, "metadata.yaml"), []byte(meta), 0o644)).To(gomega.Succeed())
		gomega.Expect(os.WriteFile(filepath.Join(dir, "main.py"), []byte(source), 0o644)).To(gomega.Succeed())

		svc := pkgfunctions.ActiveService()
		gomega.Expect(svc).NotTo(gomega.BeNil())
		applied, err := svc.ApplyDir(context.Background(), dir, "functions-bdd-user")
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(applied.Name).To(gomega.Equal(name))
		gomega.Expect(applied.Version).To(gomega.BeNumerically(">=", 1))

		hub.Default.Unregister(hub.CapFunctions)
		capability.DefaultRegistry.Unregister(hub.CapFunctions, capfunctions.OpInvoke)
		capability.DefaultRegistry.Unregister(hub.CapFunctions, capfunctions.OpGet)
		capability.DefaultRegistry.Unregister(hub.CapFunctions, capfunctions.OpHealth)
		gomega.Expect(capfunctions.Register()).To(gomega.Succeed())

		capOut, err := capability.Invoke(context.Background(), hub.CapFunctions, capfunctions.OpInvoke, map[string]any{
			"name":    name,
			"version": applied.Version,
			"event":   map[string]any{"n": 1},
		})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(capOut).NotTo(gomega.BeNil())
		data, ok := capOut.Data.(*pkgfunctions.InvokeResult)
		if !ok {
			asMap, mapOK := capOut.Data.(map[string]any)
			gomega.Expect(mapOK).To(gomega.BeTrue())
			gomega.Expect(asMap["status"]).To(gomega.Equal("succeeded"))
		} else {
			gomega.Expect(data.Status).To(gomega.Equal("succeeded"))
		}

		// management get still works with user token (HTTP secrets unused on capability path)
		getReq := functionsAuthRequest(http.MethodGet, "/service/functions/get/"+name, token, nil)
		getResp, err := App.Test(getReq)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(getResp.StatusCode).To(gomega.Equal(http.StatusOK))
	})

	It("returns management get without source or token fields", func() {
		token := functionsUserToken()
		name := "meta-fn-" + types.Id()
		meta := "name: " + name + "\nhttp:\n  auth:\n    token: t\n"
		source := "import json,sys\nprint(json.dumps({\"ok\": True}))\n"
		body, _ := sonic.Marshal(map[string]any{
			"metadata":   meta,
			"entrypoint": "main.py",
			"source":     source,
		})
		applyReq := functionsAuthRequest(http.MethodPost, "/service/functions/apply", token, body)
		applyResp, err := App.Test(applyReq)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(applyResp.StatusCode).To(gomega.Equal(http.StatusOK))

		getReq := functionsAuthRequest(http.MethodGet, "/service/functions/get/"+name, token, nil)
		getResp, err := App.Test(getReq)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(getResp.StatusCode).To(gomega.Equal(http.StatusOK))
		var getOut protocol.Response
		gomega.Expect(sonic.Unmarshal(ReadBody(getResp), &getOut)).To(gomega.Succeed())
		gomega.Expect(getOut.Status).To(gomega.Equal(protocol.Success))
		data, ok := getOut.Data.(map[string]any)
		gomega.Expect(ok).To(gomega.BeTrue())
		gomega.Expect(data).NotTo(gomega.HaveKey("source"))
		gomega.Expect(data).NotTo(gomega.HaveKey("token"))
	})
})
