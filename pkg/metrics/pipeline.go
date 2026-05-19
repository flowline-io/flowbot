package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/flowline-io/flowbot/pkg/stats"
)

// PipelineCollector holds typed metrics for pipeline execution.
// When initialized with a nil stats, all methods are no-op.
type PipelineCollector struct {
	runTotal     *prometheus.CounterVec
	runDuration  *prometheus.HistogramVec
	stepTotal    *prometheus.CounterVec
	stepDuration *prometheus.HistogramVec
	stepRetry    *prometheus.CounterVec
	resumeTotal  *prometheus.CounterVec
}

// NewPipelineCollector creates a PipelineCollector backed by stats.
// Returns a no-op collector when stats is nil.
func NewPipelineCollector(st *stats.Stats) *PipelineCollector {
	if st == nil {
		return &PipelineCollector{}
	}
	return &PipelineCollector{
		runTotal:     st.RegisterCounterVec("pipeline_run_total", "Runs by pipeline and status", "pipeline", "status"),
		runDuration:  st.RegisterHistogramVec("pipeline_run_duration_seconds", "Run duration distribution", "pipeline", "status"),
		stepTotal:    st.RegisterCounterVec("pipeline_step_total", "Steps by pipeline, step, and status", "pipeline", "step", "status"),
		stepDuration: st.RegisterHistogramVec("pipeline_step_duration_seconds", "Step duration distribution", "pipeline", "step", "capability", "status"),
		stepRetry:    st.RegisterCounterVec("pipeline_step_retry_total", "Step retry count", "pipeline", "step"),
		resumeTotal:  st.RegisterCounterVec("pipeline_resume_total", "Pipeline resume count", "pipeline"),
	}
}

// IncRunTotal increments the run counter for the given pipeline and status.
func (c *PipelineCollector) IncRunTotal(pipeline, status string) {
	if c.runTotal == nil {
		return
	}
	defer recoverLog("pipeline_run_total")
	c.runTotal.WithLabelValues(sanitizeLabel(pipeline), sanitizeLabel(status)).Inc()
}

// ObserveRunDuration records a run duration observation for the given pipeline and status.
func (c *PipelineCollector) ObserveRunDuration(pipeline, status string, seconds float64) {
	if c.runDuration == nil {
		return
	}
	defer recoverLog("pipeline_run_duration_seconds")
	c.runDuration.WithLabelValues(sanitizeLabel(pipeline), sanitizeLabel(status)).Observe(seconds)
}

// IncStepTotal increments the step counter for the given pipeline, step, and status.
func (c *PipelineCollector) IncStepTotal(pipeline, step, status string) {
	if c.stepTotal == nil {
		return
	}
	defer recoverLog("pipeline_step_total")
	c.stepTotal.WithLabelValues(sanitizeLabel(pipeline), sanitizeLabel(step), sanitizeLabel(status)).Inc()
}

// ObserveStepDuration records a step duration observation.
func (c *PipelineCollector) ObserveStepDuration(pipeline, step, capability, status string, seconds float64) {
	if c.stepDuration == nil {
		return
	}
	defer recoverLog("pipeline_step_duration_seconds")
	c.stepDuration.WithLabelValues(sanitizeLabel(pipeline), sanitizeLabel(step), sanitizeLabel(capability), sanitizeLabel(status)).Observe(seconds)
}

// IncStepRetry increments the retry counter for a given pipeline and step.
func (c *PipelineCollector) IncStepRetry(pipeline, step string) {
	if c.stepRetry == nil {
		return
	}
	defer recoverLog("pipeline_step_retry_total")
	c.stepRetry.WithLabelValues(sanitizeLabel(pipeline), sanitizeLabel(step)).Inc()
}

// IncResume increments the resume counter for the given pipeline.
func (c *PipelineCollector) IncResume(pipeline string) {
	if c.resumeTotal == nil {
		return
	}
	defer recoverLog("pipeline_resume_total")
	c.resumeTotal.WithLabelValues(sanitizeLabel(pipeline)).Inc()
}
