package metrics

import (
	"log"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/flowline-io/flowbot/pkg/stats"
)

// PipelineCollector holds typed metrics for pipeline execution.
// When initialized with a nil stats, all methods are no-op.
type PipelineCollector struct {
	runTotal      *prometheus.CounterVec
	runDuration   *prometheus.HistogramVec
	stepTotal     *prometheus.CounterVec
	stepDuration  *prometheus.HistogramVec
	stepRetry     *prometheus.CounterVec
	resumeTotal   *prometheus.CounterVec
	cronSkipTotal *prometheus.CounterVec
	cronExecTotal *prometheus.CounterVec
	cronDuration  *prometheus.HistogramVec
}

// NewPipelineCollector creates a PipelineCollector backed by stats.
// Returns a no-op collector when stats is nil or if registration fails.
func NewPipelineCollector(st *stats.Stats) *PipelineCollector {
	if st == nil {
		return &PipelineCollector{}
	}
	var err error
	c := &PipelineCollector{}
	c.runTotal, err = st.RegisterCounterVec("pipeline_run_total", "Runs by pipeline and status", "pipeline", "status")
	if err != nil {
		log.Printf("[metrics] pipeline: failed to register counter vec: %v", err)
		return &PipelineCollector{}
	}
	c.runDuration, err = st.RegisterHistogramVec("pipeline_run_duration_seconds", "Run duration distribution", "pipeline", "status")
	if err != nil {
		log.Printf("[metrics] pipeline: failed to register histogram vec: %v", err)
		return &PipelineCollector{}
	}
	c.stepTotal, err = st.RegisterCounterVec("pipeline_step_total", "Steps by pipeline, step, and status", "pipeline", "step", "status")
	if err != nil {
		log.Printf("[metrics] pipeline: failed to register counter vec: %v", err)
		return &PipelineCollector{}
	}
	c.stepDuration, err = st.RegisterHistogramVec("pipeline_step_duration_seconds", "Step duration distribution", "pipeline", "step", "capability", "status")
	if err != nil {
		log.Printf("[metrics] pipeline: failed to register histogram vec: %v", err)
		return &PipelineCollector{}
	}
	c.stepRetry, err = st.RegisterCounterVec("pipeline_step_retry_total", "Step retry count", "pipeline", "step")
	if err != nil {
		log.Printf("[metrics] pipeline: failed to register counter vec: %v", err)
		return &PipelineCollector{}
	}
	c.resumeTotal, err = st.RegisterCounterVec("pipeline_resume_total", "Pipeline resume count", "pipeline")
	if err != nil {
		log.Printf("[metrics] pipeline: failed to register counter vec: %v", err)
		return &PipelineCollector{}
	}
	c.cronSkipTotal, err = st.RegisterCounterVec("pipeline_cron_skip_total", "Cron job skip count by pipeline", "pipeline")
	if err != nil {
		log.Printf("[metrics] pipeline: failed to register counter vec: %v", err)
		return &PipelineCollector{}
	}
	c.cronExecTotal, err = st.RegisterCounterVec("pipeline_cron_exec_total", "Cron job execution count by pipeline and status", "pipeline", "status")
	if err != nil {
		log.Printf("[metrics] pipeline: failed to register counter vec: %v", err)
		return &PipelineCollector{}
	}
	c.cronDuration, err = st.RegisterHistogramVec("pipeline_cron_duration_seconds", "Cron job duration distribution", "pipeline")
	if err != nil {
		log.Printf("[metrics] pipeline: failed to register histogram vec: %v", err)
		return &PipelineCollector{}
	}
	return c
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

// IncCronSkip increments the cron skip counter for the given pipeline.
func (c *PipelineCollector) IncCronSkip(pipeline string) {
	if c.cronSkipTotal == nil {
		return
	}
	defer recoverLog("pipeline_cron_skip_total")
	c.cronSkipTotal.WithLabelValues(sanitizeLabel(pipeline)).Inc()
}

// IncCronExec increments the cron execution counter for the given pipeline and status.
func (c *PipelineCollector) IncCronExec(pipeline, status string) {
	if c.cronExecTotal == nil {
		return
	}
	defer recoverLog("pipeline_cron_exec_total")
	c.cronExecTotal.WithLabelValues(sanitizeLabel(pipeline), sanitizeLabel(status)).Inc()
}

// ObserveCronDuration records a cron job duration observation for the given pipeline.
func (c *PipelineCollector) ObserveCronDuration(pipeline string, seconds float64) {
	if c.cronDuration == nil {
		return
	}
	defer recoverLog("pipeline_cron_duration_seconds")
	c.cronDuration.WithLabelValues(sanitizeLabel(pipeline)).Observe(seconds)
}
