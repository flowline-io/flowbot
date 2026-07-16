package metrics

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
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
		c.IncCronSkip("p")
	})
}

func TestPipelineCollector_CounterMetrics(t *testing.T) {
	runTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pipeline_run_total",
			Help: "Runs by pipeline and status",
		},
		[]string{"pipeline", "status"},
	)
	stepRetry := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pipeline_step_retry_total",
			Help: "Step retry count",
		},
		[]string{"pipeline", "step"},
	)
	c := &PipelineCollector{runTotal: runTotal, stepRetry: stepRetry}

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
	runDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "pipeline_run_duration_seconds",
			Help:    "Run duration distribution",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"pipeline", "status"},
	)
	stepDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "pipeline_step_duration_seconds",
			Help:    "Step duration distribution",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"pipeline", "step", "capability", "status"},
	)
	c := &PipelineCollector{runDuration: runDuration, stepDuration: stepDuration}

	c.ObserveRunDuration("p1", "done", 2.0)
	c.ObserveRunDuration("p1", "done", 3.0)
	c.ObserveStepDuration("p1", "step1", "bookmark", "done", 0.5)
	assert.NotNil(t, c.runDuration)
	assert.NotNil(t, c.stepDuration)
}

func TestPipelineCollector_LabelsSanitized(t *testing.T) {
	runTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pipeline_run_total",
			Help: "Runs by pipeline and status",
		},
		[]string{"pipeline", "status"},
	)
	stepTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pipeline_step_total",
			Help: "Steps by pipeline, step, and status",
		},
		[]string{"pipeline", "step", "status"},
	)
	c := &PipelineCollector{runTotal: runTotal, stepTotal: stepTotal}

	c.IncRunTotal("my pipeline!", "done")
	c.IncStepTotal("p", "step with spaces", "done")

	assert.NotPanics(t, func() {
		c.IncRunTotal("name with / and ?", "done")
	})
}

func TestPipelineCollector_CronMetrics(t *testing.T) {
	cronExec := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pipeline_cron_exec_total",
			Help: "Cron executions by pipeline and status",
		},
		[]string{"pipeline", "status"},
	)
	cronDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "pipeline_cron_duration_seconds",
			Help:    "Cron duration distribution",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"pipeline"},
	)
	c := &PipelineCollector{cronExecTotal: cronExec, cronDuration: cronDuration}

	c.IncCronExec("nightly-sync", "done")
	c.IncCronExec("nightly-sync", "done")
	c.IncCronExec("nightly-sync", "failed")
	c.ObserveCronDuration("nightly-sync", 12.5)
	c.ObserveCronDuration("nightly-sync", 8.0)

	expected := `
# HELP pipeline_cron_exec_total Cron executions by pipeline and status
# TYPE pipeline_cron_exec_total counter
pipeline_cron_exec_total{pipeline="nightly-sync",status="done"} 2
pipeline_cron_exec_total{pipeline="nightly-sync",status="failed"} 1
`
	err := testutil.CollectAndCompare(cronExec, strings.NewReader(expected), "pipeline_cron_exec_total")
	assert.NoError(t, err)
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
		{name: "IncCronSkip", fn: func() { c.IncCronSkip("p") }},
		{name: "IncCronExec", fn: func() { c.IncCronExec("p", "done") }},
		{name: "ObserveCronDuration", fn: func() { c.ObserveCronDuration("p", 1.0) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, tt.fn)
		})
	}
}
