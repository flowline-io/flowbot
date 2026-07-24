//go:build integration

package specs

import (
	"context"
	"os"
	"path/filepath"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/agent/msg"
	"github.com/flowline-io/flowbot/pkg/agent/session"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/tmc/langchaingo/llms"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Chat Agent", Label("module", "chat-agent"), func() {
	var restoreSessionTitleLLM func()

	BeforeEach(func() {
		restoreSessionTitleLLM = chatagent.DisableSessionTitleLLMForTest()
	})

	AfterEach(func() {
		chatagent.WaitForSessionTitleGenerationForTest()
		if restoreSessionTitleLLM != nil {
			restoreSessionTitleLLM()
			restoreSessionTitleLLM = nil
		}
	})

	It("persists session entries and returns assistant reply using fake model", func() {
		config.App.ChatAgent.ChatModel = "fake-model"
		config.App.Models = []config.Model{
			{Provider: agentllm.ProviderOpenAI, ApiKey: "test", ModelNames: []string{"fake-model"}},
		}
		config.App.ChatAgent.Compaction = config.CompactionConfig{Auto: new(false)}
		config.App.ChatAgent.ToolModel = ""

		model := agentllm.NewFakeModel(agentllm.ResponseScript{Content: "hello from agent"})
		orig := chatagent.NewModelForTest
		chatagent.NewModelForTest = func(_ context.Context, _ string) (llms.Model, string, error) {
			return model, "fake-model", nil
		}
		defer func() { chatagent.NewModelForTest = orig }()

		ctx := context.Background()
		sessionID := "bdd-session-1-" + types.Id()
		wsDir, err := os.MkdirTemp("", "chat-agent-bdd-*")
		Expect(err).NotTo(HaveOccurred())
		config.App.ChatAgent.Workspace = wsDir
		Expect(chatagent.CreateSession(ctx, "uid-bdd", sessionID)).To(Succeed())

		svc := chatagent.NewService()
		reply, err := svc.Run(ctx, chatagent.RunRequest{SessionID: sessionID, Text: "hi"}, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(reply).To(ContainSubstring("hello from agent"))

		sess, err := store.Database.GetChatSession(ctx, sessionID)
		Expect(err).NotTo(HaveOccurred())
		Expect(sess.LeafID).NotTo(BeEmpty())

		entries, err := store.Database.ListChatSessionEntries(ctx, sessionID)
		Expect(err).NotTo(HaveOccurred())
		Expect(entries).NotTo(BeEmpty())

		Expect(svc.CloseSession(ctx, sessionID)).To(Succeed())
		closed, err := store.Database.GetChatSession(ctx, sessionID)
		Expect(err).NotTo(HaveOccurred())
		Expect(closed.State).To(Equal(int(schema.ChatSessionClosed)))
	})

	It("routes to tool model after tool execution when dual models are configured", func() {
		config.App.Models = []config.Model{
			{
				Provider:   agentllm.ProviderOpenAI,
				ApiKey:     "test",
				ModelNames: []string{"chat-model", "tool-model"},
			},
		}
		config.App.ChatAgent = config.ChatAgentConfig{
			Compaction: config.CompactionConfig{Auto: new(false)},
			ChatModel:  "chat-model",
			ToolModel:  "tool-model",
		}

		fake := agentllm.NewFakeModel(
			agentllm.ResponseScript{ToolCalls: []llms.ToolCall{{
				ID: "call-1", Type: "function",
				FunctionCall: &llms.FunctionCall{Name: "read_skill", Arguments: `{"name":"missing-skill"}`},
			}}},
			agentllm.ResponseScript{Content: "done after tool"},
		)
		orig := chatagent.NewModelForTest
		chatagent.NewModelForTest = func(_ context.Context, _ string) (llms.Model, string, error) {
			return fake, "chat-model", nil
		}
		defer func() { chatagent.NewModelForTest = orig }()

		ctx := context.Background()
		sessionID := "bdd-dual-model-" + types.Id()
		wsDir, err := os.MkdirTemp("", "chat-agent-dual-*")
		Expect(err).NotTo(HaveOccurred())
		config.App.ChatAgent.Workspace = wsDir
		Expect(chatagent.CreateSession(ctx, "uid-dual", sessionID)).To(Succeed())

		svc := chatagent.NewService()
		reply, err := svc.Run(ctx, chatagent.RunRequest{SessionID: sessionID, Text: "use a skill"}, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(reply).To(ContainSubstring("done after tool"))

		entries, err := store.Database.ListChatSessionEntries(ctx, sessionID)
		Expect(err).NotTo(HaveOccurred())
		assistantModels := make([]string, 0)
		for _, row := range entries {
			payload, err := sonic.Marshal(row.Payload)
			Expect(err).NotTo(HaveOccurred())
			entry, err := session.UnmarshalEntry(payload)
			Expect(err).NotTo(HaveOccurred())
			assistant, ok := entry.Message.(msg.AssistantMessage)
			if ok && assistant.Model != "" {
				assistantModels = append(assistantModels, assistant.Model)
			}
		}
		Expect(assistantModels).To(Equal([]string{"chat-model", "tool-model"}))

		Expect(svc.CloseSession(ctx, sessionID)).To(Succeed())
	})

	It("blocks write_file in plan mode and allows it after returning to normal", func() {
		config.App.ChatAgent.ChatModel = "fake-model"
		config.App.Models = []config.Model{
			{Provider: agentllm.ProviderOpenAI, ApiKey: "test", ModelNames: []string{"fake-model"}},
		}
		config.App.ChatAgent.Compaction = config.CompactionConfig{Auto: new(false)}
		config.App.ChatAgent.ToolModel = ""

		target := "plan-mode-target.txt"
		writeArgs := `{"path":"` + target + `","content":"updated"}`

		model := agentllm.NewFakeModel(
			agentllm.ResponseScript{ToolCalls: []llms.ToolCall{{
				ID: "call-plan", Type: "function",
				FunctionCall: &llms.FunctionCall{Name: "write_file", Arguments: writeArgs},
			}}},
			agentllm.ResponseScript{Content: "Here is the plan without making changes."},
			agentllm.ResponseScript{ToolCalls: []llms.ToolCall{{
				ID: "call-run", Type: "function",
				FunctionCall: &llms.FunctionCall{Name: "write_file", Arguments: writeArgs},
			}}},
			agentllm.ResponseScript{Content: "File updated."},
		)
		orig := chatagent.NewModelForTest
		chatagent.NewModelForTest = func(_ context.Context, _ string) (llms.Model, string, error) {
			return model, "fake-model", nil
		}

		ctx := context.Background()
		sessionID := "bdd-plan-mode-" + types.Id()
		svc := chatagent.NewService()
		defer func() {
			chatagent.WaitForSessionTitleGenerationForTest()
			svc.EvictHarnessPool(sessionID)
			chatagent.NewModelForTest = orig
		}()

		wsDir, err := os.MkdirTemp("", "chat-agent-plan-*")
		Expect(err).NotTo(HaveOccurred())
		config.App.ChatAgent.Workspace = wsDir
		Expect(chatagent.CreateSession(ctx, "uid-plan", sessionID)).To(Succeed())
		Expect(chatagent.SetSessionMode(ctx, sessionID, chatagent.ModePlan)).To(Succeed())

		reply, err := svc.Run(ctx, chatagent.RunRequest{SessionID: sessionID, Text: "edit plan-mode-target.txt"}, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(reply).To(ContainSubstring("plan"))

		_, statErr := os.Stat(filepath.Join(wsDir, target))
		Expect(os.IsNotExist(statErr)).To(BeTrue())

		chatagent.WaitForSessionTitleGenerationForTest()
		Expect(chatagent.SetSessionMode(ctx, sessionID, chatagent.ModeNormal)).To(Succeed())

		// write_file requires approval by default; register a confirm gate and
		// approve the pending request so the run can proceed, mirroring how an
		// Web UI or the HTTP SSE endpoint resolves asks.
		pub := chatagent.NewChannelPublisher(4)
		gate := chatagent.NewConfirmGate(sessionID, pub, nil)
		runState := chatagent.NewAPIRunState(pub, gate)
		Expect(svc.TrySetAPIRunState(sessionID, runState)).To(Succeed())
		defer svc.ClearAPIRunState(sessionID, runState)

		type runResult struct {
			reply string
			err   error
		}
		done := make(chan runResult, 1)
		go func() {
			r, runErr := svc.Run(ctx, chatagent.RunRequest{
				SessionID: sessionID,
				Text:      "now edit plan-mode-target.txt",
				API: &chatagent.APIRunOptions{
					Publisher: pub,
					Confirm:   gate,
				},
			}, nil)
			done <- runResult{reply: r, err: runErr}
		}()

		Eventually(pub.Events(), "5s").Should(Receive(HaveField("Type", chatagent.EventTypeConfirm)))
		_, err = svc.ResolveConfirm(sessionID, gate.ID(), true, chatagent.ConfirmModeOnce, "", chatagent.ConfirmReasonApproved)
		Expect(err).NotTo(HaveOccurred())

		var result runResult
		Eventually(done, "5s").Should(Receive(&result))
		Expect(result.err).NotTo(HaveOccurred())
		Expect(result.reply).To(ContainSubstring("updated"))

		_, statErr = os.Stat(filepath.Join(wsDir, target))
		Expect(statErr).NotTo(HaveOccurred())

		Expect(svc.CloseSession(ctx, sessionID)).To(Succeed())
	})

	It("persists plan resources and resolves plan:// and file:// links", func() {
		config.App.ChatAgent.ChatModel = "fake-model"
		config.App.Models = []config.Model{
			{Provider: agentllm.ProviderOpenAI, ApiKey: "test", ModelNames: []string{"fake-model"}},
		}
		config.App.ChatAgent.Compaction = config.CompactionConfig{Auto: new(false)}
		config.App.ChatAgent.ToolModel = ""

		model := agentllm.NewFakeModel(agentllm.ResponseScript{Content: "# Resource Plan\nStep one"})
		orig := chatagent.NewModelForTest
		chatagent.NewModelForTest = func(_ context.Context, _ string) (llms.Model, string, error) {
			return model, "fake-model", nil
		}
		defer func() { chatagent.NewModelForTest = orig }()

		ctx := context.Background()
		sessionID := "bdd-resource-" + types.Id()
		wsDir, err := os.MkdirTemp("", "chat-agent-resource-*")
		Expect(err).NotTo(HaveOccurred())
		config.App.ChatAgent.Workspace = wsDir
		notePath := filepath.Join(wsDir, "note.txt")
		Expect(os.WriteFile(notePath, []byte("file body"), 0o644)).To(Succeed())
		Expect(chatagent.CreateSession(ctx, "uid-resource", sessionID)).To(Succeed())
		Expect(chatagent.SetSessionMode(ctx, sessionID, chatagent.ModePlan)).To(Succeed())

		svc := chatagent.NewService()
		reply, err := svc.Run(ctx, chatagent.RunRequest{SessionID: sessionID, Text: "draft plan"}, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(reply).To(ContainSubstring("plan://"))

		plans, err := chatagent.ListPlanSummaries(ctx, sessionID)
		Expect(err).NotTo(HaveOccurred())
		Expect(plans).NotTo(BeEmpty())

		planContent, err := svc.ResolveResource(ctx, sessionID, plans[0].URI)
		Expect(err).NotTo(HaveOccurred())
		Expect(planContent.Content).To(ContainSubstring("Resource Plan"))

		fileContent, err := svc.ResolveResource(ctx, sessionID, "file://note.txt")
		Expect(err).NotTo(HaveOccurred())
		Expect(fileContent.Content).To(Equal("file body"))

		_, err = svc.ResolveResource(ctx, sessionID, "file://../outside.txt")
		Expect(err).To(HaveOccurred())

		Expect(svc.CloseSession(ctx, sessionID)).To(Succeed())
	})
})
