//go:build integration

package specs

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/llm"
)

var _ = Describe("LLM Provider", Label("module", "llm", "smoke"), func() {

	BeforeEach(func() {
		llm.RegisterOpenAI()
		llm.RegisterGemini()
		llm.RegisterAnthropic()
	})

	Describe("provider factory", func() {
		DescribeTable("creates correct provider type",
			func(providerType, apiKey, baseURL string) {
				p, err := llm.NewProvider(config.Model{
					Provider: providerType,
					ApiKey:   apiKey,
					BaseUrl:  baseURL,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(p.Name()).NotTo(BeEmpty())
			},
			Entry("openai", "openai", "sk-test", ""),
			Entry("openai compatible", "openai_compatible", "sk-test", "http://localhost:8080"),
			Entry("gemini", "gemini", "gk-test", ""),
			Entry("anthropic", "anthropic", "ak-test", ""),
		)

		DescribeTable("rejects invalid provider types",
			func(providerType string) {
				_, err := llm.NewProvider(config.Model{
					Provider: providerType,
					ApiKey:   "test-key",
				})
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(llm.ErrUnknownProvider))
			},
			Entry("empty string", ""),
			Entry("unknown value", "unknown_provider"),
			Entry("misspelled", "open-ai"),
		)
	})

	Describe("http error handling", func() {
		DescribeTable("returns error for non-200 status codes",
			func(providerType string, statusCode int) {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(statusCode)
				}))
				defer srv.Close()

				p, err := llm.NewProvider(config.Model{
					Provider: providerType,
					ApiKey:   "test-key",
					BaseUrl:  srv.URL,
				})
				Expect(err).NotTo(HaveOccurred())

				_, err = p.Generate(suiteCtx, &llm.GenerateRequest{
					Model:    "test-model",
					Messages: []*llm.Message{{Role: llm.UserRole, Content: "hi"}},
				})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("%d", statusCode)))
			},
			Entry("openai 401", "openai_compatible", http.StatusUnauthorized),
			Entry("openai 429", "openai_compatible", http.StatusTooManyRequests),
			Entry("openai 500", "openai_compatible", http.StatusInternalServerError),
			Entry("gemini 400", "gemini", http.StatusBadRequest),
			Entry("gemini 500", "gemini", http.StatusInternalServerError),
			Entry("anthropic 401", "anthropic", http.StatusUnauthorized),
			Entry("anthropic 429", "anthropic", http.StatusTooManyRequests),
		)
	})
})
