package metrics

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/stats"
)

func TestNewWorkflowCollector(t *testing.T) {
	t.Run("returns no-op when stats is nil", func(t *testing.T) {
		c := NewWorkflowCollector(nil)
		assert.NotNil(t, c)
		c.IncRunTotal("w", "done")
		c.ObserveRunDuration("w", "done", 2.0)
		c.IncStepTotal("w", "s", "done")
		c.ObserveStepDuration("w", "s", "capability", "done", 0.5)
		c.IncStepRetry("w", "s")
		c.IncResume("w")
		c.SetConcurrency("w", 3)
	})
}

func TestWorkflowCollector_CounterMetrics(t *testing.T) {
	stats.Init(&stats.MetricsConfig{PushGatewayURL: "http://localhost:9091", PushInterval: 60})
	s := stats.NewStats()
	c := NewWorkflowCollector(s)

	c.IncRunTotal("archive-workflow", "done")
	c.IncRunTotal("archive-workflow", "done")
	c.IncRunTotal("archive-workflow", "failed")

	c.IncStepRetry("archive-workflow", "task1")
	c.IncStepRetry("archive-workflow", "task1")
	c.IncStepRetry("archive-workflow", "task2")

	c.IncResume("archive-workflow")

	expected := `
# HELP workflow_run_total Runs by workflow and status
# TYPE workflow_run_total counter
workflow_run_total{status="done",workflow="archive-workflow"} 2
workflow_run_total{status="failed",workflow="archive-workflow"} 1
`
	err := testutil.CollectAndCompare(c.runTotal, strings.NewReader(expected), "workflow_run_total")
	assert.NoError(t, err)
}

func TestWorkflowCollector_ConcurrencyGauge(t *testing.T) {
	stats.Init(&stats.MetricsConfig{PushGatewayURL: "http://localhost:9091", PushInterval: 60})
	s := stats.NewStats()
	c := NewWorkflowCollector(s)
	c.SetConcurrency("dag-workflow", 3)
	c.SetConcurrency("dag-workflow", 0)
	assert.NotNil(t, c.concurrency)
}

func TestWorkflowCollector_NoopMethodsDontPanic(t *testing.T) {
	c := NewWorkflowCollector(nil)
	tests := []struct {
		name string
		fn   func()
	}{
		{name: "IncRunTotal", fn: func() { c.IncRunTotal("w", "done") }},
		{name: "ObserveRunDuration", fn: func() { c.ObserveRunDuration("w", "done", 1.0) }},
		{name: "IncStepTotal", fn: func() { c.IncStepTotal("w", "s", "done") }},
		{name: "SetConcurrency", fn: func() { c.SetConcurrency("w", 5) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, tt.fn)
		})
	}
}
