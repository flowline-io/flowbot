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
})
