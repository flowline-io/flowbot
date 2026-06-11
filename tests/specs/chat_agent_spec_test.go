//go:build integration

package specs

import (
	"context"
	"os"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/tmc/langchaingo/llms"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Chat Agent", Label("module", "chat-agent"), func() {
	It("persists session entries and returns assistant reply using fake model", func() {
		config.App.Agents = []config.Agent{
			{Name: "chat", Enabled: true, Model: "fake-model"},
		}
		config.App.Models = []config.Model{
			{Provider: agentllm.ProviderOpenAI, ApiKey: "test", ModelNames: []string{"fake-model"}, ContextWindows: map[string]int{"fake-model": 128000}},
		}
		config.App.ChatAgent.Compaction = config.CompactionConfig{Enabled: false}

		model := agentllm.NewFakeModel(agentllm.ResponseScript{Content: "hello from agent"})
		orig := chatagent.NewModelForTest
		chatagent.NewModelForTest = func(_ context.Context, _ string) (llms.Model, string, error) {
			return model, "fake-model", nil
		}
		defer func() { chatagent.NewModelForTest = orig }()

		ctx := context.Background()
		sessionID := "bdd-session-1"
		wsDir, err := os.MkdirTemp("", "chat-agent-bdd-*")
		Expect(err).NotTo(HaveOccurred())
		config.App.ChatAgent.Workspace = wsDir
		Expect(chatagent.CreateSession(ctx, "uid-bdd", sessionID)).To(Succeed())

		svc := chatagent.NewService()
		reply, err := svc.Run(ctx, chatagent.RunRequest{SessionID: sessionID, Text: "hi"})
		Expect(err).NotTo(HaveOccurred())
		Expect(reply).To(ContainSubstring("hello from agent"))

		sess, err := store.Database.GetChatSession(ctx, sessionID)
		Expect(err).NotTo(HaveOccurred())
		Expect(sess.LeafID).NotTo(BeEmpty())

		entries, err := store.Database.ListChatSessionEntries(ctx, sessionID)
		Expect(err).NotTo(HaveOccurred())
		Expect(entries).NotTo(BeEmpty())

		Expect(chatagent.CloseSession(ctx, sessionID)).To(Succeed())
		closed, err := store.Database.GetChatSession(ctx, sessionID)
		Expect(err).NotTo(HaveOccurred())
		Expect(closed.State).To(Equal(int(schema.ChatSessionClosed)))
	})
})
