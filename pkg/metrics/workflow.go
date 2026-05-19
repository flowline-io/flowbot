package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/flowline-io/flowbot/pkg/stats"
)

// WorkflowCollector holds typed metrics for workflow execution.
// When initialized with a nil stats, all methods are no-op.
type WorkflowCollector struct {
	runTotal     *prometheus.CounterVec
	runDuration  *prometheus.HistogramVec
	stepTotal    *prometheus.CounterVec
	stepDuration *prometheus.HistogramVec
	stepRetry    *prometheus.CounterVec
	resumeTotal  *prometheus.CounterVec
	concurrency  *prometheus.GaugeVec
}

// NewWorkflowCollector creates a WorkflowCollector backed by stats.
// Returns a no-op collector when stats is nil.
func NewWorkflowCollector(st *stats.Stats) *WorkflowCollector {
	if st == nil {
		return &WorkflowCollector{}
	}
	return &WorkflowCollector{
		runTotal:     st.RegisterCounterVec("workflow_run_total", "Runs by workflow and status", "workflow", "status"),
		runDuration:  st.RegisterHistogramVec("workflow_run_duration_seconds", "Run duration distribution", "workflow", "status"),
		stepTotal:    st.RegisterCounterVec("workflow_step_total", "Steps by workflow, step, and status", "workflow", "step", "status"),
		stepDuration: st.RegisterHistogramVec("workflow_step_duration_seconds", "Step duration distribution", "workflow", "step", "action_type", "status"),
		stepRetry:    st.RegisterCounterVec("workflow_step_retry_total", "Step retry count", "workflow", "step"),
		resumeTotal:  st.RegisterCounterVec("workflow_resume_total", "Workflow resume count", "workflow"),
		concurrency:  st.RegisterGaugeVec("workflow_concurrency_gauge", "Running tasks in DAG parallel mode", "workflow"),
	}
}

// IncRunTotal increments the run counter for the given workflow and status.
func (c *WorkflowCollector) IncRunTotal(workflow, status string) {
	if c.runTotal == nil {
		return
	}
	defer recoverLog("workflow_run_total")
	c.runTotal.WithLabelValues(sanitizeLabel(workflow), sanitizeLabel(status)).Inc()
}

// ObserveRunDuration records a run duration observation for the given workflow and status.
func (c *WorkflowCollector) ObserveRunDuration(workflow, status string, seconds float64) {
	if c.runDuration == nil {
		return
	}
	defer recoverLog("workflow_run_duration_seconds")
	c.runDuration.WithLabelValues(sanitizeLabel(workflow), sanitizeLabel(status)).Observe(seconds)
}

// IncStepTotal increments the step counter for the given workflow, step, and status.
func (c *WorkflowCollector) IncStepTotal(workflow, step, status string) {
	if c.stepTotal == nil {
		return
	}
	defer recoverLog("workflow_step_total")
	c.stepTotal.WithLabelValues(sanitizeLabel(workflow), sanitizeLabel(step), sanitizeLabel(status)).Inc()
}

// ObserveStepDuration records a step duration observation.
func (c *WorkflowCollector) ObserveStepDuration(workflow, step, actionType, status string, seconds float64) {
	if c.stepDuration == nil {
		return
	}
	defer recoverLog("workflow_step_duration_seconds")
	c.stepDuration.WithLabelValues(sanitizeLabel(workflow), sanitizeLabel(step), sanitizeLabel(actionType), sanitizeLabel(status)).Observe(seconds)
}

// IncStepRetry increments the retry counter for a given workflow and step.
func (c *WorkflowCollector) IncStepRetry(workflow, step string) {
	if c.stepRetry == nil {
		return
	}
	defer recoverLog("workflow_step_retry_total")
	c.stepRetry.WithLabelValues(sanitizeLabel(workflow), sanitizeLabel(step)).Inc()
}

// IncResume increments the resume counter for the given workflow.
func (c *WorkflowCollector) IncResume(workflow string) {
	if c.resumeTotal == nil {
		return
	}
	defer recoverLog("workflow_resume_total")
	c.resumeTotal.WithLabelValues(sanitizeLabel(workflow)).Inc()
}

// SetConcurrency sets the concurrency gauge for the given workflow.
func (c *WorkflowCollector) SetConcurrency(workflow string, count int) {
	if c.concurrency == nil {
		return
	}
	defer recoverLog("workflow_concurrency_gauge")
	c.concurrency.WithLabelValues(sanitizeLabel(workflow)).Set(float64(count))
}
