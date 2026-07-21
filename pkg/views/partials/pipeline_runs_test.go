package partials

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
)

func TestPipelineStepRunsDetail(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		steps    []*gen.PipelineStepRun
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
			steps: []*gen.PipelineStepRun{
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
			},
		},
		{
			name: "step with no Params and no Result renders non-clickable row",
			steps: []*gen.PipelineStepRun{
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
			steps: []*gen.PipelineStepRun{
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
			steps: []*gen.PipelineStepRun{
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
			name: "data-testid step-row present on summary rows for all steps",
			steps: []*gen.PipelineStepRun{
				{
					StepName:  "alpha",
					Status:    2,
					Attempt:   1,
					StartedAt: now,
				},
				{
					StepName:  "beta",
					Params:    map[string]any{"x": 1},
					Status:    2,
					Attempt:   1,
					StartedAt: now,
				},
				{
					StepName:  "gamma",
					Params:    map[string]any{"y": 2},
					Result:    map[string]any{"z": 3},
					Status:    2,
					Attempt:   1,
					StartedAt: now,
				},
			},
			contains: []string{
				`data-testid="step-row-alpha"`,
				`data-testid="step-row-beta"`,
				`data-testid="step-row-gamma"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := PipelineStepRunsDetail(tt.steps).Render(context.Background(), &buf)
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
