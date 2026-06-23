//go:build integration

package specs

import (
	"context"
	"os"
	"time"

	"github.com/flowline-io/flowbot/internal/server/chatagent"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/tmc/langchaingo/llms"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Chat Agent Scheduled Tasks", Label("module", "chat-agent", "scheduled-tasks"), func() {
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

	It("executes a one-shot task in an isolated session and marks it completed", func() {
		model := agentllm.NewFakeModel(agentllm.ResponseScript{Content: "scheduled reply"})
		orig := chatagent.NewModelForTest
		chatagent.NewModelForTest = func(_ context.Context, _ string) (llms.Model, string, error) {
			return model, "fake-model", nil
		}
		defer func() { chatagent.NewModelForTest = orig }()

		ctx := context.Background()
		wsDir, err := os.MkdirTemp("", "chat-agent-sched-*")
		Expect(err).NotTo(HaveOccurred())
		config.App.ChatAgent = config.ChatAgentConfig{
			ChatModel:  "fake-model",
			Workspace:  wsDir,
			Compaction: config.CompactionConfig{Auto: new(false)},
		}
		config.App.Models = []config.Model{
			{Provider: agentllm.ProviderOpenAI, ApiKey: "test", ModelNames: []string{"fake-model"}},
		}

		taskFlag := "bdd-task-once-" + types.Id()
		runAt := time.Now().UTC().Add(-time.Minute)
		task := &gen.ChatScheduledTask{
			Flag:            taskFlag,
			UID:             "uid-sched",
			Name:            "once job",
			ScheduleKind:    string(schema.ChatScheduledTaskKindOnce),
			Prompt:          "run scheduled prompt",
			RunAt:           &runAt,
			SourceSessionID: "source-session",
			State:           string(schema.ChatScheduledTaskStateActive),
		}
		Expect(store.Database.CreateChatScheduledTask(ctx, task)).To(Succeed())

		chatagent.ExecuteScheduledTaskForTest(ctx, task)

		updated, err := store.Database.GetChatScheduledTask(ctx, task.Flag)
		Expect(err).NotTo(HaveOccurred())
		Expect(updated.State).To(Equal(string(schema.ChatScheduledTaskStateCompleted)))

		runs, err := store.Database.ListChatScheduledTaskRuns(ctx, task.Flag, 5)
		Expect(err).NotTo(HaveOccurred())
		Expect(runs).NotTo(BeEmpty())
		Expect(runs[0].Reply).To(ContainSubstring("scheduled reply"))
	})
})
