//go:build integration

package specs

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/config"
)

var _ = Describe("Agent LLM", Label("module", "llm", "smoke"), func() {

	Describe("model factory", func() {
		DescribeTable("creates model for configured providers",
			func(providerType, modelName, apiKey, baseURL string) {
				config.App.Models = []config.Model{{
					Provider:   providerType,
					ModelNames: []string{modelName},
					ApiKey:     apiKey,
					BaseUrl:    baseURL,
				}}

				model, resolved, err := llm.NewModel(context.Background(), modelName)
				Expect(err).NotTo(HaveOccurred())
				Expect(model).NotTo(BeNil())
				Expect(resolved).To(Equal(modelName))
			},
			Entry("openai", "openai", "gpt-test", "sk-test", ""),
			Entry("openai compatible", "openai_compatible", "local-model", "sk-test", "http://localhost:8080"),
			Entry("gemini", "gemini", "gemini-test", "gk-test", ""),
			Entry("anthropic", "anthropic", "claude-test", "ak-test", ""),
		)

		DescribeTable("rejects unknown models",
			func(modelName string) {
				config.App.Models = nil
				_, _, err := llm.NewModel(context.Background(), modelName)
				Expect(err).To(HaveOccurred())
			},
			Entry("empty string", ""),
			Entry("unknown value", "unknown-model"),
			Entry("misspelled", "gpt-99"),
		)
	})

	Describe("agent config lookup", func() {
		BeforeEach(func() {
			config.App.Agents = []config.Agent{
				{Name: llm.AgentExtractTags, Model: "gpt-test", Enabled: true},
				{Name: llm.AgentSimilarTags, Model: "gpt-test", Enabled: false},
			}
		})

		It("returns model for enabled agents", func() {
			Expect(llm.AgentModelName(llm.AgentExtractTags)).To(Equal("gpt-test"))
			Expect(llm.AgentEnabled(llm.AgentExtractTags)).To(BeTrue())
		})

		It("returns empty for disabled agents", func() {
			Expect(llm.AgentModelName(llm.AgentSimilarTags)).To(BeEmpty())
			Expect(llm.AgentEnabled(llm.AgentSimilarTags)).To(BeFalse())
		})
	})
})
