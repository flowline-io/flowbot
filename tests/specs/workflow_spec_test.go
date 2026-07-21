//go:build integration
// +build integration

package specs

import (
	"context"
	"net/http"
	"time"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Workflow Module", Label("module", "workflow"), func() {

	Describe("Webservice — apply / list / run", func() {
		It("rejects apply without yaml", func() {
			body, _ := sonic.Marshal(map[string]any{})
			req := JSONRequest(http.MethodPost, "/service/workflow/apply", body)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Or(
				Equal(http.StatusBadRequest),
				Equal(http.StatusUnauthorized),
				Equal(http.StatusServiceUnavailable),
			))
		})

		It("rejects run without workflow name", func() {
			body, _ := sonic.Marshal(map[string]any{"input": map[string]any{}})
			req := JSONRequest(http.MethodPost, "/service/workflow/run", body)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Or(
				Equal(http.StatusBadRequest),
				Equal(http.StatusUnauthorized),
				Equal(http.StatusServiceUnavailable),
			))
		})

		It("lists workflows or reports unavailable", func() {
			req := JSONRequest(http.MethodGet, "/service/workflow/list", nil)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Or(
				Equal(http.StatusOK),
				Equal(http.StatusUnauthorized),
				Equal(http.StatusServiceUnavailable),
			))
		})
	})

	Describe("Workflow type system", func() {
		It("creates workflow metadata", func() {
			meta := types.WorkflowMetadata{
				Name:    "test-workflow",
				Resumable: true,
			Triggers: []types.WorkflowTriggerDef{
					{Type: "cron", Rule: types.KV{"schedule": "*/5 * * * *"}},
				},
				Tasks: []types.WorkflowTask{
					{
						ID:     "task-1",
						Action: "http.request",
						Params: types.KV{"url": "https://example.com"},
					},
				},
			}
			Expect(meta.Name).To(Equal("test-workflow"))
			Expect(meta.Resumable).To(BeTrue())
			Expect(len(meta.Tasks)).To(Equal(1))
		})

		It("creates workflow task with retry config", func() {
			task := types.WorkflowTask{
				ID:     "retry-task",
				Action: "notify.send",
				Retry: &types.RetryConfig{
					MaxAttempts: 3,
					Delay:       1 * time.Second,
					Backoff:     types.BackoffExponential,
					MaxDelay:    30 * time.Second,
					Jitter:      true,
				},
			}
			Expect(task.Retry.ToBackoffConfig().MaxAttempts).To(BeNumerically(">", 1))
			Expect(task.Retry.MaxAttempts).To(Equal(3))
			Expect(task.Retry.Backoff).To(Equal(types.BackoffExponential))
		})

		It("validates retry config", func() {
			cfg := types.RetryConfig{MaxAttempts: 0}
			Expect(cfg.ToBackoffConfig().MaxAttempts > 1).To(BeFalse())

			cfg2 := types.RetryConfig{MaxAttempts: 1}
			Expect(cfg2.ToBackoffConfig().MaxAttempts > 1).To(BeFalse())

			cfg3 := types.RetryConfig{MaxAttempts: 2}
			Expect(cfg3.ToBackoffConfig().MaxAttempts > 1).To(BeTrue())
		})

		It("has correct backoff constants", func() {
			Expect(types.BackoffFixed).To(Equal("fixed"))
			Expect(types.BackoffLinear).To(Equal("linear"))
			Expect(types.BackoffExponential).To(Equal("exponential"))
		})
	})

	Describe("Workflow task states", func() {
		It("has all standard task states", func() {
			Expect(string(types.TaskStatePending)).To(Equal("PENDING"))
			Expect(string(types.TaskStateRunning)).To(Equal("RUNNING"))
			Expect(string(types.TaskStateCompleted)).To(Equal("COMPLETED"))
			Expect(string(types.TaskStateFailed)).To(Equal("FAILED"))
			Expect(string(types.TaskStateCancelled)).To(Equal("CANCELED"))
		})

		It("detects active task states", func() {
			Expect(types.TaskStatePending.IsActive()).To(BeTrue())
			Expect(types.TaskStateRunning.IsActive()).To(BeTrue())
			Expect(types.TaskStateCompleted.IsActive()).To(BeFalse())
			Expect(types.TaskStateFailed.IsActive()).To(BeFalse())
		})
	})

	Describe("Protocol error handling", func() {
		It("has protocol-level error builders", func() {
			resp := protocol.NewFailedResponse(types.ErrNotFound)
			Expect(resp.Status).To(Equal(protocol.Failed))
		})

		It("creates success response", func() {
			resp := protocol.NewSuccessResponse(map[string]string{"result": "ok"})
			Expect(resp.Status).To(Equal(protocol.Success))
		})
	})

	Describe("Context operations", func() {
		It("creates and uses runtime context", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			Expect(ctx.Err()).NotTo(HaveOccurred())
		})
	})

	Describe("Workflow Parallel Execution", func() {
		It("has MaxConcurrency field on metadata", func() {
			meta := types.WorkflowMetadata{
				Name:           "parallel-test",
				MaxConcurrency: 3,
				Pipeline:       []string{"a", "b"},
				Tasks: []types.WorkflowTask{
					{ID: "a", Action: "mapper:"},
					{ID: "b", Action: "mapper:", Conn: []string{"a"}},
				},
			}
			Expect(meta.MaxConcurrency).To(Equal(3))
		})

		It("defaults MaxConcurrency to zero", func() {
			meta := types.WorkflowMetadata{
				Name:     "sequential-test",
				Pipeline: []string{"a"},
				Tasks:    []types.WorkflowTask{{ID: "a", Action: "mapper:"}},
			}
			Expect(meta.MaxConcurrency).To(Equal(0))
		})

		It("supports parallel execution via Conn dependencies", func() {
			tasks := []types.WorkflowTask{
				{ID: "a", Conn: []string{"b", "c"}},
				{ID: "b", Conn: []string{"d"}},
				{ID: "c", Conn: []string{"d"}},
				{ID: "d"},
			}
			Expect(tasks).To(HaveLen(4))
			Expect(tasks[0].Conn).To(ConsistOf("b", "c"))
			Expect(tasks[3].Conn).To(BeEmpty())
		})
	})
})
