package partials

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
)

func TestWorkflowStepRunsDetail(t *testing.T) {
	t.Parallel()
	now := time.Now()
	done := now.Add(time.Second)
	tests := []struct {
		name     string
		steps    []*gen.WorkflowStepRun
		contains []string
		excludes []string
	}{
		{
			name:  "empty steps renders empty message",
			steps: nil,
			contains: []string{
				`data-testid="workflow-step-runs-empty"`,
				"No task runs recorded for this run.",
			},
		},
		{
			name: "step with params and result is expandable",
			steps: []*gen.WorkflowStepRun{
				{
					StepID:      "echo",
					StepName:    "echo",
					Action:      "mapper:",
					ActionType:  "mapper",
					Params:      map[string]any{"msg": "hi"},
					Result:      map[string]any{"out": "hi"},
					Status:      int(schema.WorkflowRunDone),
					Attempt:     1,
					StartedAt:   now,
					CompletedAt: &done,
				},
			},
			contains: []string{
				`data-testid="workflow-step-runs-detail"`,
				`data-testid="workflow-step-row-echo"`,
				"step-chevron",
				`data-testid="workflow-step-detail-row-echo"`,
				"Input",
				"Output",
				"mapper:",
				"run-json-preview",
				`data-testid="workflow-step-output-json"`,
			},
		},
		{
			name: "step without params or result is not expandable",
			steps: []*gen.WorkflowStepRun{
				{
					StepID:     "noop",
					StepName:   "noop",
					Action:     "shell:true",
					ActionType: "shell",
					Status:     int(schema.WorkflowRunDone),
					Attempt:    1,
					StartedAt:  now,
				},
			},
			contains: []string{
				`data-testid="workflow-step-row-noop"`,
				"shell:true",
			},
			excludes: []string{
				"step-chevron",
				"workflow-step-detail-row-noop",
			},
		},
		{
			name: "failed step with error only is expandable and open",
			steps: []*gen.WorkflowStepRun{
				{
					StepID:     "fail",
					StepName:   "fail",
					Action:     "shell:false",
					ActionType: "shell",
					Status:     int(schema.WorkflowRunFailed),
					Error:      "exit status 1",
					Attempt:    1,
					StartedAt:  now,
				},
			},
			contains: []string{
				`data-testid="workflow-step-row-fail"`,
				"step-chevron",
				"rotate-90",
				`data-testid="workflow-step-detail-row-fail"`,
				`data-testid="workflow-step-error-fail"`,
				`data-testid="run-error-summary"`,
				`data-testid="run-waterfall"`,
				"Error",
				"exit status 1",
			},
			excludes: []string{
				`class="step-detail-row hidden"`,
			},
		},
		{
			name: "failed step shows error text",
			steps: []*gen.WorkflowStepRun{
				{
					StepID:     "fail",
					StepName:   "fail",
					Action:     "shell:false",
					ActionType: "shell",
					Params:     map[string]any{"cmd": "false"},
					Status:     int(schema.WorkflowRunFailed),
					Error:      "exit 1",
					Attempt:    2,
					StartedAt:  now,
				},
			},
			contains: []string{
				"exit 1",
				"Failed",
				`data-testid="workflow-step-error-fail"`,
				"rotate-90",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := WorkflowStepRunsDetail(tt.steps).Render(context.Background(), &buf)
			require.NoError(t, err)
			html := buf.String()
			for _, sub := range tt.contains {
				assert.Contains(t, html, sub)
			}
			for _, sub := range tt.excludes {
				assert.NotContains(t, html, sub)
			}
		})
	}
}

func TestWorkflowStepRunDuration(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	done := now.Add(1500 * time.Millisecond)
	tests := []struct {
		name string
		sr   *gen.WorkflowStepRun
		want string
	}{
		{name: "nil", sr: nil, want: "-"},
		{name: "incomplete", sr: &gen.WorkflowStepRun{StartedAt: now}, want: "-"},
		{name: "completed", sr: &gen.WorkflowStepRun{StartedAt: now, CompletedAt: &done}, want: "2s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, WorkflowStepRunDuration(tt.sr))
		})
	}
}
