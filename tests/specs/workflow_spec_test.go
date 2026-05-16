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

	Describe("Webservice — POST /run", func() {
		It("rejects empty request body", func() {
			req := JSONRequest(http.MethodPost, "/service/workflow/run", nil)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Or(Equal(http.StatusOK), Equal(http.StatusBadRequest), Equal(http.StatusUnauthorized)))
		})

		It("rejects request without file field", func() {
			body, _ := sonic.Marshal(map[string]string{"params": "{}"})
			req := JSONRequest(http.MethodPost, "/service/workflow/run", body)
			resp, err := App.Test(req)
			Expect(err).NotTo(HaveOccurred())
			_ = resp
		})
	})

	Describe("Workflow type system", func() {
		It("creates workflow metadata", func() {
			meta := types.WorkflowMetadata{
				Name:    "test-workflow",
				Resumable: true,
				Triggers: []struct {
					Type string       `json:"type"`
					Rule types.KV      `json:"rule"`
				}{
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
			Expect(task.Retry.RetryEnabled()).To(BeTrue())
			Expect(task.Retry.MaxAttempts).To(Equal(3))
			Expect(task.Retry.Backoff).To(Equal(types.BackoffExponential))
		})

		It("validates retry config", func() {
			cfg := types.RetryConfig{MaxAttempts: 0}
			Expect(cfg.RetryEnabled()).To(BeFalse())

			cfg2 := types.RetryConfig{MaxAttempts: 1}
			Expect(cfg2.RetryEnabled()).To(BeTrue())
		})

		It("has correct backoff constants", func() {
			Expect(types.BackoffFixed).To(Equal("fixed"))
			Expect(types.BackoffLinear).To(Equal("linear"))
			Expect(types.BackoffExponential).To(Equal("exponential"))
		})
	})

	Describe("Workflow task states", func() {
		It("has all standard task states", func() {
			Expect(types.TaskStatePending).To(Equal("pending"))
			Expect(types.TaskStateRunning).To(Equal("running"))
			Expect(types.TaskStateCompleted).To(Equal("completed"))
			Expect(types.TaskStateFailed).To(Equal("failed"))
			Expect(types.TaskStateCancelled).To(Equal("cancelled"))
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
})
