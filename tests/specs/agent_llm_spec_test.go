//go:build integration
// +build integration

package specs

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"
	"github.com/tmc/langchaingo/llms"

	"github.com/flowline-io/flowbot/internal/server"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var agentLLMRoutesOnce sync.Once

func mountAgentLLMRoutes(app *fiber.App) {
	agentLLMRoutesOnce.Do(func() {
		server.RegisterAgentLLMRoutes(app)
	})
}

func agentHeadlessToken() string {
	token, err := auth.NewToken()
	Expect(err).NotTo(HaveOccurred())
	bddSeedAccessToken(token, "agent-headless-bdd", []string{auth.ScopeAgentHeadless})
	return token
}

var _ = Describe("Agent LLM proxy API", Label("agent", "api"), func() {
	BeforeEach(func() {
		mountAgentLLMRoutes(App)
	})

	It("rejects completions without agent:headless scope", func() {
		body := `{"model":"x","messages":[{"role":"user","content":"hi"}]}`
		req := JSONRequest(http.MethodPost, "/agent/v1/chat/completions", []byte(body))
		resp, err := App.Test(req)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
	})

	It("forces server chat model for authorized callers", func() {
		origChat := config.App.ChatAgent
		config.App.ChatAgent.ChatModel = "bdd-forced-model"
		DeferCleanup(func() { config.App.ChatAgent = origChat })

		origModels := config.App.Models
		config.App.Models = []config.Model{{
			Provider:   "openai_compatible",
			ModelNames: []string{"bdd-forced-model"},
			ApiKey:     "sk-test",
			BaseUrl:    "http://127.0.0.1:9/v1",
		}}
		DeferCleanup(func() { config.App.Models = origModels })

		agentllm.ResetModelPoolForTest()
		DeferCleanup(agentllm.ResetModelPoolForTest)
		fake := agentllm.NewFakeModel(agentllm.ResponseScript{Content: "bdd-ok"})
		agentllm.SetModelCreatorForTest(func(_ context.Context, modelName string) (llms.Model, string, error) {
			Expect(modelName).To(Equal("bdd-forced-model"))
			return fake, modelName, nil
		})
		DeferCleanup(func() { agentllm.SetModelCreatorForTest(nil) })

		token := agentHeadlessToken()
		body := `{"model":"client-ignored","messages":[{"role":"user","content":"hi"}],"stream":false}`
		req := JSONRequest(http.MethodPost, "/agent/v1/chat/completions", []byte(body))
		req.Header.Set("X-AccessToken", token)
		resp, err := App.Test(req, 10*time.Second)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		raw, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(strings.Contains(string(raw), "bdd-ok")).To(BeTrue())
		var out map[string]any
		Expect(sonic.Unmarshal(raw, &out)).To(Succeed())
		Expect(out["model"]).To(Equal("bdd-forced-model"))
	})
})
