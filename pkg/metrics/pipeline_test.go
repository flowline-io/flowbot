package metrics

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/stats"
)

func TestNewPipelineCollector(t *testing.T) {
	t.Run("returns no-op when stats is nil", func(t *testing.T) {
		c := NewPipelineCollector(nil)
		assert.NotNil(t, c)
		c.IncRunTotal("p", "done")
		c.ObserveRunDuration("p", "done", 1.5)
		c.IncStepTotal("p", "s", "done")
		c.ObserveStepDuration("p", "s", "cap", "done", 0.5)
		c.IncStepRetry("p", "s")
		c.IncResume("p")
	})
}

func TestPipelineCollector_CounterMetrics(t *testing.T) {
	stats.Init(&stats.MetricsConfig{PushGatewayURL: "http://localhost:9091", PushInterval: 60})
	s := stats.NewStats()
	c := NewPipelineCollector(s)

	c.IncRunTotal("archive-items", "done")
	c.IncRunTotal("archive-items", "done")
	c.IncRunTotal("archive-items", "cancel")
	c.IncRunTotal("sync-bookmarks", "done")

	c.IncStepRetry("archive-items", "step1")
	c.IncStepRetry("archive-items", "step1")

	c.IncResume("archive-items")

	expected := `
# HELP pipeline_run_total Runs by pipeline and status
# TYPE pipeline_run_total counter
pipeline_run_total{pipeline="archive-items",status="done"} 2
pipeline_run_total{pipeline="archive-items",status="cancel"} 1
pipeline_run_total{pipeline="sync-bookmarks",status="done"} 1
`
	err := testutil.CollectAndCompare(c.runTotal, strings.NewReader(expected), "pipeline_run_total")
	assert.NoError(t, err)
}

func TestPipelineCollector_HistogramMetrics(t *testing.T) {
	stats.Init(&stats.MetricsConfig{PushGatewayURL: "http://localhost:9091", PushInterval: 60})
	s := stats.NewStats()
	c := NewPipelineCollector(s)

	c.ObserveRunDuration("p1", "done", 2.0)
	c.ObserveRunDuration("p1", "done", 3.0)
	c.ObserveStepDuration("p1", "step1", "bookmark", "done", 0.5)
	assert.NotNil(t, c.runDuration)
	assert.NotNil(t, c.stepDuration)
}

func TestPipelineCollector_LabelsSanitized(t *testing.T) {
	stats.Init(&stats.MetricsConfig{PushGatewayURL: "http://localhost:9091", PushInterval: 60})
	s := stats.NewStats()
	c := NewPipelineCollector(s)

	c.IncRunTotal("my pipeline!", "done")
	c.IncStepTotal("p", "step with spaces", "done")

	assert.NotPanics(t, func() {
		c.IncRunTotal("name with / and ?", "done")
	})
}

func TestPipelineCollector_NoopMethodsDontPanic(t *testing.T) {
	c := NewPipelineCollector(nil)
	tests := []struct {
		name string
		fn   func()
	}{
		{name: "IncRunTotal", fn: func() { c.IncRunTotal("p", "done") }},
		{name: "ObserveRunDuration", fn: func() { c.ObserveRunDuration("p", "done", 1.0) }},
		{name: "IncStepTotal", fn: func() { c.IncStepTotal("p", "s", "done") }},
		{name: "ObserveStepDuration", fn: func() { c.ObserveStepDuration("p", "s", "c", "done", 1.0) }},
		{name: "IncStepRetry", fn: func() { c.IncStepRetry("p", "s") }},
		{name: "IncResume", fn: func() { c.IncResume("p") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, tt.fn)
		})
	}
}
