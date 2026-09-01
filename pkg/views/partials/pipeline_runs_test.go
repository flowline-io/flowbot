package partials

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/i18n"
	"github.com/flowline-io/flowbot/pkg/types/model"
)

func TestPipelineStepRunsDetail(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		steps    []model.PipelineStepRun
		retry    PipelineRunRetry
		contains []string
		excludes []string
	}{
		{
			name:  "empty steps slice renders no step runs message",
			steps: nil,
			contains: []string{
				"No step runs recorded for this run.",
			},
		},
		{
			name: "step with both Params and Result renders clickable row with chevron and detail row with Input and Output",
			steps: []model.PipelineStepRun{
				{
					StepName:  "build",
					Params:    map[string]any{"source": "main.go"},
					Result:    map[string]any{"binary": "app"},
					Status:    2,
					Attempt:   1,
					StartedAt: now,
				},
			},
			contains: []string{
				`data-testid="step-row-build"`,
				"cursor-pointer",
				"step-chevron",
				"data-step-toggle",
				`data-testid="step-detail-row-build"`,
				"<details ",
				"Input",
				"Output",
				`flowbot-table-expand-cell`,
			},
		},
		{
			name: "step with no Params and no Result renders non-clickable row",
			steps: []model.PipelineStepRun{
				{
					StepName:  "noop",
					Status:    2,
					Attempt:   1,
					StartedAt: now,
				},
			},
			contains: []string{
				`data-testid="step-row-noop"`,
			},
			excludes: []string{
				"cursor-pointer",
				"step-chevron",
				"data-step-toggle",
			},
		},
		{
			name: "step with Params only renders clickable row with Input details and empty Output",
			steps: []model.PipelineStepRun{
				{
					StepName:  "fetch",
					Params:    map[string]any{"url": "https://example.com"},
					Status:    2,
					Attempt:   1,
					StartedAt: now,
				},
			},
			contains: []string{
				`data-testid="step-row-fetch"`,
				"cursor-pointer",
				"step-chevron",
				"data-step-toggle",
				"<details ",
				"Input",
				"Output: (empty)",
			},
		},
		{
			name: "step with Result only renders clickable row with Output details and empty Input",
			steps: []model.PipelineStepRun{
				{
					StepName:  "deploy",
					Result:    map[string]any{"url": "https://app.example.com"},
					Status:    2,
					Attempt:   1,
					StartedAt: now,
				},
			},
			contains: []string{
				`data-testid="step-row-deploy"`,
				"cursor-pointer",
				"step-chevron",
				"data-step-toggle",
				"<details ",
				"Input: (empty)",
				"Output",
			},
		},
		{
			name: "failed step defaults open with error summary and error block",
			steps: []model.PipelineStepRun{
				{
					StepName:  "ok",
					Status:    2,
					Attempt:   1,
					StartedAt: now,
				},
				{
					StepName:  "boom",
					Status:    4,
					Error:     "timeout exceeded",
					Attempt:   1,
					StartedAt: now,
					Params:    map[string]any{"x": 1},
				},
			},
			contains: []string{
				`data-testid="run-error-summary"`,
				"timeout exceeded",
				`data-testid="step-error-boom"`,
				"rotate-90",
				`data-testid="run-waterfall"`,
			},
			excludes: []string{
				`data-testid="retry-failed-run"`,
			},
		},
		{
			name: "failed run retry button uses shared confirm attributes",
			steps: []model.PipelineStepRun{
				{
					StepName:  "boom",
					Status:    4,
					Error:     "kanboard create task",
					Attempt:   1,
					StartedAt: now,
					Params:    map[string]any{"title": "x"},
				},
			},
			retry: PipelineRunRetry{PipelineName: "new-bookmark-task", RunID: 73, Enabled: true},
			contains: []string{
				`data-testid="retry-failed-run"`,
				`hx-post="/service/web/pipelines/new-bookmark-task/runs/73/retry"`,
				`data-confirm=`,
				`data-confirm-title=`,
				`data-confirm-btn=`,
				`data-confirm-class="btn-warning"`,
			},
		},
		{
			name: "successful expandable step stays collapsed",
			steps: []model.PipelineStepRun{
				{
					StepName:  "build",
					Params:    map[string]any{"source": "main.go"},
					Result:    map[string]any{"binary": "app"},
					Status:    2,
					Attempt:   1,
					StartedAt: now,
				},
			},
			contains: []string{
				`data-testid="step-detail-row-build"`,
				"step-detail-row hidden",
			},
			excludes: []string{
				`data-testid="run-error-summary"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := PipelineStepRunsDetail(i18n.DefaultContext(), tt.steps, tt.retry).Render(i18n.DefaultContext(), &buf)
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}
			output := buf.String()
			for _, s := range tt.contains {
				if !strings.Contains(output, s) {
					t.Errorf("output should contain %q", s)
				}
			}
			for _, s := range tt.excludes {
				if strings.Contains(output, s) {
					t.Errorf("output should not contain %q", s)
				}
			}
		})
	}
}

func TestSprintJSON(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    map[string]any
		contains []string
		excludes []string
	}{
		{
			name:  "empty map",
			input: nil,
		},
		{
			name:     "expands nested json string",
			input:    map[string]any{"result": `{"capability":"karakeep","operation":"list"}`},
			contains: []string{`"capability": "karakeep"`, `"operation": "list"`},
			excludes: []string{`\"capability\"`},
		},
		{
			name:     "keeps plain string",
			input:    map[string]any{"msg": "hello"},
			contains: []string{`"msg": "hello"`},
		},
		{
			name:     "expands nested array json",
			input:    map[string]any{"payload": `[{"id":"a"}]`},
			contains: []string{`"id": "a"`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := sprintJSON(tt.input)
			if tt.input == nil {
				if got != "" {
					t.Fatalf("got %q, want empty", got)
				}
				return
			}
			for _, sub := range tt.contains {
				if !strings.Contains(got, sub) {
					t.Errorf("sprintJSON() missing %q in %q", sub, got)
				}
			}
			for _, sub := range tt.excludes {
				if strings.Contains(got, sub) {
					t.Errorf("sprintJSON() should not contain %q in %q", sub, got)
				}
			}
		})
	}
}

func TestPipelineRunsTable_expandConstrainsCellWithoutScroll(t *testing.T) {
	t.Parallel()
	runs := []model.PipelineRun{{
		ID:        7,
		EventID:   "evt-7",
		StartedAt: time.Date(2026, 8, 31, 8, 6, 7, 0, time.UTC),
		CreatedAt: time.Date(2026, 8, 31, 8, 6, 7, 0, time.UTC),
	}}
	var buf bytes.Buffer
	if err := PipelineRunsTable(i18n.DefaultContext(), "demo", runs).Render(i18n.DefaultContext(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	assertExpandRowStaysInPlace(t, buf.String(),
		`hx-target="#steps-7 td"`,
		`hx-swap="innerHTML show:none"`,
		`id="steps-7"`,
		`class="run-detail-row"`,
		`flowbot-table-expand-cell`,
	)
}

func TestRunsDuration(t *testing.T) {
	t.Parallel()
	created := time.Date(2026, 8, 16, 2, 44, 22, 0, time.UTC)
	started := time.Date(2026, 8, 28, 19, 40, 0, 0, time.UTC)
	done := started.Add(250 * time.Millisecond)
	tests := []struct {
		name string
		run  model.PipelineRun
		want string
	}{
		{name: "incomplete", run: model.PipelineRun{StartedAt: started, CreatedAt: created}, want: "-"},
		{
			name: "uses started_at not created_at after retry",
			run:  model.PipelineRun{StartedAt: started, CreatedAt: created, CompletedAt: &done},
			want: "250ms",
		},
		{
			name: "falls back to created_at when started_at zero",
			run: model.PipelineRun{
				CreatedAt:   created,
				CompletedAt: new(created.Add(2 * time.Second)),
			},
			want: "2s",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := runsDuration(tt.run); got != tt.want {
				t.Fatalf("runsDuration() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStepRunsDuration(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 1, 14, 0, 0, 0, time.UTC)
	done := now.Add(13 * time.Millisecond)
	tests := []struct {
		name string
		sr   model.PipelineStepRun
		want string
	}{
		{name: "incomplete", sr: model.PipelineStepRun{StartedAt: now}, want: "-"},
		{name: "completed", sr: model.PipelineStepRun{StartedAt: now, CompletedAt: &done}, want: "13ms"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := stepRunsDuration(tt.sr); got != tt.want {
				t.Fatalf("stepRunsDuration() = %q, want %q", got, tt.want)
			}
		})
	}
}

//go:fix inline
func ptrTime(t time.Time) *time.Time { return new(t) }
