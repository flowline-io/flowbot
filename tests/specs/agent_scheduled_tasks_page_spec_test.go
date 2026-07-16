//go:build integration
// +build integration

package specs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	webmod "github.com/flowline-io/flowbot/internal/modules/web"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/chatscheduledtask"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/chatscheduledtaskrun"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	pkgconfig "github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
)

type agentScheduledTasksWebAdapter struct {
	store.Adapter
	ent *gen.Client
	uid string
}

func (a *agentScheduledTasksWebAdapter) Open(_ pkgconfig.StoreType) error { return nil }
func (a *agentScheduledTasksWebAdapter) Close() error                     { return nil }
func (a *agentScheduledTasksWebAdapter) IsOpen() bool                     { return true }
func (a *agentScheduledTasksWebAdapter) GetName() string                  { return "bdd-agent-scheduled-tasks" }
func (a *agentScheduledTasksWebAdapter) Stats() any                       { return nil }
func (a *agentScheduledTasksWebAdapter) GetDB() any                       { return a.ent }

func (a *agentScheduledTasksWebAdapter) ParameterGet(_ context.Context, flag string) (gen.Parameter, error) {
	return gen.Parameter{
		ID:   1,
		Flag: flag,
		Params: map[string]any{
			"uid":    a.uid,
			"topic":  "test",
			"scopes": []string{"admin:*"},
		},
		ExpiredAt: time.Now().Add(time.Hour),
	}, nil
}

func (a *agentScheduledTasksWebAdapter) ListChatScheduledTasks(ctx context.Context, opts store.ListChatScheduledTasksOptions) ([]*gen.ChatScheduledTask, error) {
	q := a.ent.ChatScheduledTask.Query().
		Order(
			gen.Desc(chatscheduledtask.FieldUpdatedAt),
			gen.Desc(chatscheduledtask.FieldID),
		)
	if opts.UID != "" {
		q = q.Where(chatscheduledtask.UIDEQ(opts.UID))
	}
	if len(opts.States) > 0 {
		q = q.Where(chatscheduledtask.StateIn(opts.States...))
	}
	return q.All(ctx)
}

func (a *agentScheduledTasksWebAdapter) GetChatScheduledTaskForUID(ctx context.Context, flag, uid string) (*gen.ChatScheduledTask, error) {
	row, err := a.ent.ChatScheduledTask.Query().
		Where(
			chatscheduledtask.FlagEQ(flag),
			chatscheduledtask.UIDEQ(uid),
		).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, err
	}
	return row, nil
}

func (a *agentScheduledTasksWebAdapter) ListChatScheduledTaskRuns(ctx context.Context, taskID string, limit int) ([]*gen.ChatScheduledTaskRun, error) {
	q := a.ent.ChatScheduledTaskRun.Query().
		Where(chatscheduledtaskrun.TaskIDEQ(taskID)).
		Order(
			gen.Desc(chatscheduledtaskrun.FieldStartedAt),
			gen.Desc(chatscheduledtaskrun.FieldID),
		)
	if limit > 0 {
		q = q.Limit(limit)
	}
	return q.All(ctx)
}

var _ = Describe("Agent Scheduled Tasks UI", Label("module", "web"), func() {
	var (
		origDB        store.Adapter
		origChatModel string
		adapter       *agentScheduledTasksWebAdapter
		taskID        string
		runID         string
	)

	BeforeEach(func() {
		origDB = store.Database
		origChatModel = pkgconfig.App.ChatAgent.ChatModel
		pkgconfig.App.ChatAgent.ChatModel = "bdd-test-model"

		taskID = "bdd-task-" + types.Id()
		runID = "bdd-run-" + types.Id()

		adapter = &agentScheduledTasksWebAdapter{
			ent: EntClient,
			uid: "bdd-agent-scheduled-tasks-" + types.Id(),
		}
		store.Database = adapter

		conf := json.RawMessage(`{"enabled":true,"auth":{"username":"admin","password":"flowbot-dev-pass"}}`)
		_ = webmod.InitForE2E(conf)
		webmod.MountForE2E(App)

		ctx := context.Background()
		runAt := time.Now().UTC().Add(30 * time.Minute)
		nextRunAt := time.Now().UTC().Add(45 * time.Minute)
		finishedAt := time.Now().UTC().Add(-5 * time.Minute)

		EntClient.ChatScheduledTask.Create().
			SetFlag(taskID).
			SetUID(adapter.uid).
			SetName("BDD Scheduled Task").
			SetScheduleKind(string(schema.ChatScheduledTaskKindCron)).
			SetCron("0 * * * *").
			SetPrompt("generate a report").
			SetState(string(schema.ChatScheduledTaskStateActive)).
			SetRunAt(runAt).
			SetNextRunAt(nextRunAt).
			SetCreatedAt(time.Now().Add(-time.Hour)).
			SetUpdatedAt(time.Now()).
			SaveX(ctx)

		EntClient.ChatScheduledTaskRun.Create().
			SetFlag(runID).
			SetTaskID(taskID).
			SetRunSessionID("bdd-session-" + types.Id()).
			SetState(string(schema.ChatScheduledTaskRunStateCompleted)).
			SetReply("report sent").
			SetStartedAt(time.Now().Add(-10 * time.Minute)).
			SetFinishedAt(finishedAt).
			SaveX(ctx)
	})

	AfterEach(func() {
		ctx := context.Background()
		EntClient.ChatScheduledTaskRun.Delete().Where(chatscheduledtaskrun.TaskIDEQ(taskID)).ExecX(ctx)
		EntClient.ChatScheduledTask.Delete().Where(chatscheduledtask.FlagEQ(taskID)).ExecX(ctx)
		store.Database = origDB
		pkgconfig.App.ChatAgent.ChatModel = origChatModel
	})

	Describe("GET /service/web/agent-scheduled-tasks", func() {
		It("renders the scheduled tasks list page", func() {
			req := MakeRequest(http.MethodGet, "/service/web/agent-scheduled-tasks", nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "bdd-agent-scheduled-tasks-token"})
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			body := ReadBody(resp)
			Expect(string(body)).To(ContainSubstring("Scheduled Tasks"))
			Expect(string(body)).To(ContainSubstring("BDD Scheduled Task"))
			Expect(string(body)).To(ContainSubstring(taskID))
		})
	})

	Describe("GET /service/web/agent-scheduled-tasks/:id", func() {
		It("renders scheduled task detail with runs", func() {
			req := MakeRequest(http.MethodGet, fmt.Sprintf("/service/web/agent-scheduled-tasks/%s", taskID), nil)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: "bdd-agent-scheduled-tasks-token"})
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			body := ReadBody(resp)
			Expect(string(body)).To(ContainSubstring("generate a report"))
			Expect(string(body)).To(ContainSubstring(runID))
			Expect(string(body)).To(ContainSubstring("report sent"))
		})
	})
})
