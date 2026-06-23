//go:build integration

package specs

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/internal/server/chatagent"
	agentllm "github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/tmc/langchaingo/llms"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Chat Agent Chat API", Label("module", "chat-agent", "chat"), func() {
	var (
		ctx                   context.Context
		token                 string
		uid                   types.Uid
		restoreSessionTitleLLM func()
	)

	AfterEach(func() {
		chatagent.WaitForSessionTitleGenerationForTest()
		if restoreSessionTitleLLM != nil {
			restoreSessionTitleLLM()
			restoreSessionTitleLLM = nil
		}
	})

	BeforeEach(func() {
		restoreSessionTitleLLM = chatagent.DisableSessionTitleLLMForTest()
		ctx = context.Background()
		uid = types.Uid("chat-api-bdd-user-" + types.Id())
		mountChatAgentRoutes(App)

		wsDir, err := os.MkdirTemp("", "chat-agent-chat-bdd-*")
		Expect(err).NotTo(HaveOccurred())

		config.App.ChatAgent = config.ChatAgentConfig{
			ChatModel:  "fake-model",
			Workspace:  wsDir,
			Compaction: config.CompactionConfig{Auto: new(false)},
		}
		config.App.Models = []config.Model{
			{
				Provider:   agentllm.ProviderOpenAI,
				ApiKey:     "test",
				ModelNames: []string{"fake-model"},
			},
		}

		orig := chatagent.NewModelForTest
		chatagent.NewModelForTest = func(_ context.Context, _ string) (llms.Model, string, error) {
			return agentllm.NewFakeModel(agentllm.ResponseScript{Content: "hello from chat api"}), "fake-model", nil
		}
		DeferCleanup(func() { chatagent.NewModelForTest = orig })

		token = createChatAgentAccessToken(ctx, uid)
	})

	It("creates a session, streams done SSE, and hydrates history", func() {
		createReq := chatAgentRequest(http.MethodPost, "/chatagent/sessions", token, nil)
		createResp, err := App.Test(createReq)
		Expect(err).NotTo(HaveOccurred())
		Expect(createResp.StatusCode).To(Equal(http.StatusCreated))

		createBody := ReadBody(createResp)
		var created struct {
			SessionID string `json:"session_id"`
		}
		Expect(sonic.Unmarshal(createBody, &created)).To(Succeed())
		Expect(created.SessionID).NotTo(BeEmpty())

		sendBody, err := sonic.Marshal(map[string]string{"text": "hi"})
		Expect(err).NotTo(HaveOccurred())
		sendReq := chatAgentRequest(http.MethodPost, "/chatagent/sessions/"+created.SessionID+"/messages", token, sendBody)
		sendResp, err := App.Test(sendReq)
		Expect(err).NotTo(HaveOccurred())
		Expect(sendResp.StatusCode).To(Equal(http.StatusOK))

		events := parseSSEBody(ReadBody(sendResp))
		Expect(events).NotTo(BeEmpty())

		var sawDone bool
		for _, ev := range events {
			if ev.Type == "done" {
				sawDone = true
				Expect(ev.Text).To(ContainSubstring("hello from chat api"))
			}
		}
		Expect(sawDone).To(BeTrue())

		historyReq := chatAgentRequest(http.MethodGet, "/chatagent/sessions/"+created.SessionID+"/messages", token, nil)
		historyResp, err := App.Test(historyReq)
		Expect(err).NotTo(HaveOccurred())
		Expect(historyResp.StatusCode).To(Equal(http.StatusOK))

		var history struct {
			Messages []struct {
				Role string `json:"role"`
				Text string `json:"text"`
			} `json:"messages"`
		}
		Expect(sonic.Unmarshal(ReadBody(historyResp), &history)).To(Succeed())
		Expect(history.Messages).NotTo(BeEmpty())

		var roles []string
		for _, msg := range history.Messages {
			roles = append(roles, msg.Role)
		}
		Expect(roles).To(ContainElements("user", "assistant"))
	})

	It("accepts cancel while a run is in flight", func() {
		stall := newStallModel()
		orig := chatagent.NewModelForTest
		chatagent.NewModelForTest = func(_ context.Context, _ string) (llms.Model, string, error) {
			return stall, "fake-model", nil
		}
		DeferCleanup(func() { chatagent.NewModelForTest = orig })

		createReq := chatAgentRequest(http.MethodPost, "/chatagent/sessions", token, nil)
		createResp, err := App.Test(createReq)
		Expect(err).NotTo(HaveOccurred())
		Expect(createResp.StatusCode).To(Equal(http.StatusCreated))

		var created struct {
			SessionID string `json:"session_id"`
		}
		Expect(sonic.Unmarshal(ReadBody(createResp), &created)).To(Succeed())

		sendBody, err := sonic.Marshal(map[string]string{"text": "block me"})
		Expect(err).NotTo(HaveOccurred())
		sendReq := chatAgentRequest(http.MethodPost, "/chatagent/sessions/"+created.SessionID+"/messages", token, sendBody)

		sendDone := make(chan []byte, 1)
		go func() {
			sendResp, testErr := App.Test(sendReq)
			Expect(testErr).NotTo(HaveOccurred())
			sendDone <- ReadBody(sendResp)
		}()

		Eventually(stall.Started, "5s").Should(BeClosed())

		cancelReq := chatAgentRequest(http.MethodPost, "/chatagent/sessions/"+created.SessionID+"/cancel", token, nil)
		cancelResp, cancelErr := App.Test(cancelReq)
		Expect(cancelErr).NotTo(HaveOccurred())
		Expect(cancelResp.StatusCode).To(Equal(http.StatusNoContent))

		body := <-sendDone
		events := parseSSEBody(body)
		var sawCanceled bool
		for _, ev := range events {
			if ev.Type == "canceled" {
				sawCanceled = true
			}
		}
		Expect(sawCanceled).To(BeTrue())
	})

	It("lists active sessions for the authenticated user", func() {
		var firstID string
		for range 2 {
			createReq := chatAgentRequest(http.MethodPost, "/chatagent/sessions", token, nil)
			createResp, err := App.Test(createReq)
			Expect(err).NotTo(HaveOccurred())
			Expect(createResp.StatusCode).To(Equal(http.StatusCreated))

			var created struct {
				SessionID string `json:"session_id"`
			}
			Expect(sonic.Unmarshal(ReadBody(createResp), &created)).To(Succeed())
			Expect(created.SessionID).NotTo(BeEmpty())
			if firstID == "" {
				firstID = created.SessionID
			}
		}

		listReq := chatAgentRequest(http.MethodGet, "/chatagent/sessions", token, nil)
		listResp, err := App.Test(listReq)
		Expect(err).NotTo(HaveOccurred())
		Expect(listResp.StatusCode).To(Equal(http.StatusOK))

		var listed struct {
			Sessions []struct {
				SessionID string `json:"session_id"`
				State     string `json:"state"`
			} `json:"sessions"`
		}
		Expect(sonic.Unmarshal(ReadBody(listResp), &listed)).To(Succeed())
		Expect(listed.Sessions).To(HaveLen(2))

		ids := make([]string, 0, len(listed.Sessions))
		for _, sess := range listed.Sessions {
			ids = append(ids, sess.SessionID)
			Expect(sess.State).To(Equal("active"))
		}
		Expect(ids).To(ContainElement(firstID))
	})

	It("returns default permissions for the authenticated user", func() {
		getReq := chatAgentRequest(http.MethodGet, "/chatagent/permissions", token, nil)
		getResp, err := App.Test(getReq)
		Expect(err).NotTo(HaveOccurred())
		Expect(getResp.StatusCode).To(Equal(http.StatusOK))

		var view struct {
			Effective map[string]any `json:"effective"`
		}
		Expect(sonic.Unmarshal(ReadBody(getResp), &view)).To(Succeed())
		Expect(view.Effective).To(HaveKey("bash"))
	})

	It("gets and updates session mode over HTTP", func() {
		createReq := chatAgentRequest(http.MethodPost, "/chatagent/sessions", token, nil)
		createResp, err := App.Test(createReq)
		Expect(err).NotTo(HaveOccurred())
		Expect(createResp.StatusCode).To(Equal(http.StatusCreated))

		var created struct {
			SessionID string `json:"session_id"`
		}
		Expect(sonic.Unmarshal(ReadBody(createResp), &created)).To(Succeed())

		getReq := chatAgentRequest(http.MethodGet, "/chatagent/sessions/"+created.SessionID+"/mode", token, nil)
		getResp, err := App.Test(getReq)
		Expect(err).NotTo(HaveOccurred())
		Expect(getResp.StatusCode).To(Equal(http.StatusOK))

		var current struct {
			Mode string `json:"mode"`
		}
		Expect(sonic.Unmarshal(ReadBody(getResp), &current)).To(Succeed())
		Expect(current.Mode).To(Equal(chatagent.ModeNormal))

		putBody, err := sonic.Marshal(map[string]string{"mode": chatagent.ModePlan})
		Expect(err).NotTo(HaveOccurred())
		putReq := chatAgentRequest(http.MethodPut, "/chatagent/sessions/"+created.SessionID+"/mode", token, putBody)
		putResp, err := App.Test(putReq)
		Expect(err).NotTo(HaveOccurred())
		Expect(putResp.StatusCode).To(Equal(http.StatusOK))

		var updated struct {
			Mode string `json:"mode"`
		}
		Expect(sonic.Unmarshal(ReadBody(putResp), &updated)).To(Succeed())
		Expect(updated.Mode).To(Equal(chatagent.ModePlan))
	})
})

// stallModel blocks GenerateContent until the run context is canceled.
type stallModel struct {
	mu      sync.Mutex
	started chan struct{}
}

func newStallModel() *stallModel {
	return &stallModel{started: make(chan struct{})}
}

func (m *stallModel) Started() chan struct{} {
	return m.started
}

func (m *stallModel) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	resp, err := m.GenerateContent(ctx, []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeHuman, prompt)}, options...)
	if err != nil {
		return "", err
	}
	if resp == nil || len(resp.Choices) == 0 {
		return "", fmt.Errorf("stall model: empty response")
	}
	return resp.Choices[0].Content, nil
}

func (m *stallModel) GenerateContent(ctx context.Context, _ []llms.MessageContent, _ ...llms.CallOption) (*llms.ContentResponse, error) {
	m.mu.Lock()
	select {
	case <-m.started:
	default:
		close(m.started)
	}
	m.mu.Unlock()

	<-ctx.Done()
	return nil, ctx.Err()
}
