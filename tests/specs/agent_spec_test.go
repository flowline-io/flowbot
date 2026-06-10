//go:build integration

package specs

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/agent"
	"github.com/flowline-io/flowbot/pkg/agent/example/echo"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/agent/tool"
	"github.com/tmc/langchaingo/llms"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Agent Core", Label("module", "agent"), func() {
	It("runs prompt to tool to final response using fake model", func() {
		model := agentllm.NewFakeModel(
			agentllm.ResponseScript{ToolCalls: []llms.ToolCall{{
				ID: "call-1", Type: "function",
				FunctionCall: &llms.FunctionCall{Name: "echo", Arguments: `{"text":"spec"}`},
			}}},
			agentllm.ResponseScript{Content: "finished"},
		)
		reg := tool.NewRegistry()
		Expect(reg.Register(echo.Tool{})).To(Succeed())

		ag := agent.NewAgent(agent.Options{
			Model:    model,
			Registry: reg,
			Config: agent.Config{
				ModelName: "fake",
				MaxSteps:  10,
			},
		})

		stream, err := ag.Prompt(context.Background(), agent.NewUserMessage("echo please"))
		Expect(err).NotTo(HaveOccurred())

		result, err := stream.Await(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Err).NotTo(HaveOccurred())
		Expect(ag.State().Messages).NotTo(BeEmpty())
		Expect(model.Calls()).To(BeNumerically(">=", 2))
	})
})
