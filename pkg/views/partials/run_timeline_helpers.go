package partials

import (
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
)

// RunWaterfallBar is one bar in the simplified run step waterfall.
type RunWaterfallBar struct {
	// Name is the step display name.
	Name string
	// Status is the step run status code.
	Status int
	// OffsetPct is the left offset as a percent of the total timeline.
	OffsetPct float64
	// WidthPct is the bar width as a percent of the total timeline.
	WidthPct float64
}

// RunErrorSummaryItem is one failed step shown in the run error summary.
type RunErrorSummaryItem struct {
	// Name is the step display name.
	Name string
	// Error is the truncated error message.
	Error string
}

// runWaterfallInput is the minimal timing data needed to build waterfall bars.
type runWaterfallInput struct {
	Name        string
	Status      int
	StartedAt   time.Time
	CompletedAt *time.Time
}

// buildRunWaterfall computes relative offset/width bars for step runs.
// now is used as the end time for incomplete steps.
func buildRunWaterfall(steps []runWaterfallInput, now time.Time) []RunWaterfallBar {
	if len(steps) == 0 {
		return nil
	}
	minStart, maxEnd := waterfallBounds(steps, now)
	if minStart.IsZero() || maxEnd.IsZero() || !maxEnd.After(minStart) {
		return waterfallFullWidthBars(steps)
	}
	total := maxEnd.Sub(minStart).Seconds()
	if total <= 0 {
		total = 1
	}
	out := make([]RunWaterfallBar, 0, len(steps))
	for _, s := range steps {
		out = append(out, waterfallBarForStep(s, minStart, total, now))
	}
	return out
}

func waterfallBounds(steps []runWaterfallInput, now time.Time) (time.Time, time.Time) {
	var minStart, maxEnd time.Time
	for _, s := range steps {
		if s.StartedAt.IsZero() {
			continue
		}
		if minStart.IsZero() || s.StartedAt.Before(minStart) {
			minStart = s.StartedAt
		}
		end := waterfallEnd(s, now)
		if maxEnd.IsZero() || end.After(maxEnd) {
			maxEnd = end
		}
	}
	return minStart, maxEnd
}

func waterfallEnd(s runWaterfallInput, now time.Time) time.Time {
	if s.CompletedAt != nil {
		return *s.CompletedAt
	}
	return now
}

func waterfallFullWidthBars(steps []runWaterfallInput) []RunWaterfallBar {
	out := make([]RunWaterfallBar, 0, len(steps))
	for _, s := range steps {
		out = append(out, RunWaterfallBar{Name: s.Name, Status: s.Status, OffsetPct: 0, WidthPct: 100})
	}
	return out
}

func waterfallBarForStep(s runWaterfallInput, minStart time.Time, totalSec float64, now time.Time) RunWaterfallBar {
	start := s.StartedAt
	if start.IsZero() {
		start = minStart
	}
	end := waterfallEnd(s, now)
	if end.Before(start) {
		end = start
	}
	offset := start.Sub(minStart).Seconds() / totalSec * 100
	width := end.Sub(start).Seconds() / totalSec * 100
	if width < 1 {
		width = 1
	}
	if offset+width > 100 {
		width = 100 - offset
	}
	return RunWaterfallBar{
		Name:      s.Name,
		Status:    s.Status,
		OffsetPct: offset,
		WidthPct:  width,
	}
}

// PipelineStepWaterfall builds waterfall bars from pipeline step runs.
func PipelineStepWaterfall(steps []*gen.PipelineStepRun) []RunWaterfallBar {
	inputs := make([]runWaterfallInput, 0, len(steps))
	for _, s := range steps {
		if s == nil {
			continue
		}
		inputs = append(inputs, runWaterfallInput{
			Name:        s.StepName,
			Status:      s.Status,
			StartedAt:   s.StartedAt,
			CompletedAt: s.CompletedAt,
		})
	}
	return buildRunWaterfall(inputs, time.Now())
}

// WorkflowStepWaterfall builds waterfall bars from workflow step runs.
func WorkflowStepWaterfall(steps []*gen.WorkflowStepRun) []RunWaterfallBar {
	inputs := make([]runWaterfallInput, 0, len(steps))
	for _, s := range steps {
		if s == nil {
			continue
		}
		inputs = append(inputs, runWaterfallInput{
			Name:        stepDisplayName(s),
			Status:      s.Status,
			StartedAt:   s.StartedAt,
			CompletedAt: s.CompletedAt,
		})
	}
	return buildRunWaterfall(inputs, time.Now())
}

// PipelineStepErrorSummary returns failed/errored pipeline steps for the top summary.
func PipelineStepErrorSummary(steps []*gen.PipelineStepRun) []RunErrorSummaryItem {
	var out []RunErrorSummaryItem
	for _, s := range steps {
		if s == nil {
			continue
		}
		if s.Error == "" && schema.PipelineState(s.Status) != schema.PipelineFailed {
			continue
		}
		errText := s.Error
		if errText == "" {
			errText = "failed"
		}
		out = append(out, RunErrorSummaryItem{
			Name:  s.StepName,
			Error: TruncateErrorSummary(errText),
		})
	}
	return out
}

// WorkflowStepErrorSummary returns failed/errored workflow steps for the top summary.
func WorkflowStepErrorSummary(steps []*gen.WorkflowStepRun) []RunErrorSummaryItem {
	var out []RunErrorSummaryItem
	for _, s := range steps {
		if s == nil {
			continue
		}
		if s.Error == "" && schema.WorkflowRunState(s.Status) != schema.WorkflowRunFailed {
			continue
		}
		errText := s.Error
		if errText == "" {
			errText = "failed"
		}
		out = append(out, RunErrorSummaryItem{
			Name:  stepDisplayName(s),
			Error: TruncateErrorSummary(errText),
		})
	}
	return out
}

const errorSummaryMaxRunes = 160

// TruncateErrorSummary shortens long error text for the summary strip.
func TruncateErrorSummary(s string) string {
	if s == "" {
		return ""
	}
	if utf8.RuneCountInString(s) <= errorSummaryMaxRunes {
		return s
	}
	runes := []rune(s)
	return string(runes[:errorSummaryMaxRunes]) + "…"
}

// pipelineStepHasDetail reports whether a pipeline step should render an expandable detail row.
func pipelineStepHasDetail(s *gen.PipelineStepRun) bool {
	if s == nil {
		return false
	}
	return len(s.Params) > 0 || len(s.Result) > 0 || s.Error != "" || schema.PipelineState(s.Status) == schema.PipelineFailed
}

// pipelineStepDetailOpen reports whether the detail row should start expanded (failed steps).
func pipelineStepDetailOpen(s *gen.PipelineStepRun) bool {
	if s == nil {
		return false
	}
	return s.Error != "" || schema.PipelineState(s.Status) == schema.PipelineFailed
}

// PipelineWaterfallBarClass returns CSS classes for a pipeline step waterfall bar.
func PipelineWaterfallBarClass(status int) string {
	switch schema.PipelineState(status) {
	case schema.PipelineDone:
		return "run-waterfall-bar run-waterfall-bar-success"
	case schema.PipelineFailed:
		return "run-waterfall-bar run-waterfall-bar-error"
	case schema.PipelineStart:
		return "run-waterfall-bar run-waterfall-bar-running"
	default:
		return "run-waterfall-bar run-waterfall-bar-muted"
	}
}

// WorkflowWaterfallBarClass returns CSS classes for a workflow step waterfall bar.
func WorkflowWaterfallBarClass(status int) string {
	switch schema.WorkflowRunState(status) {
	case schema.WorkflowRunDone:
		return "run-waterfall-bar run-waterfall-bar-success"
	case schema.WorkflowRunFailed:
		return "run-waterfall-bar run-waterfall-bar-error"
	case schema.WorkflowRunRunning:
		return "run-waterfall-bar run-waterfall-bar-running"
	default:
		return "run-waterfall-bar run-waterfall-bar-muted"
	}
}

// WaterfallBarStyle returns the inline style for offset and width.
func WaterfallBarStyle(bar RunWaterfallBar) string {
	return fmt.Sprintf("left: %.2f%%; width: %.2f%%", bar.OffsetPct, bar.WidthPct)
}
