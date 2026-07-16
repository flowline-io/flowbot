package metrics

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/stats"
)

func TestAgentCollector_CounterMetrics(t *testing.T) {
	runTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "agent_run_total_test"},
		[]string{"status"},
	)
	llmRequest := prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "agent_llm_request_total_test"},
		[]string{"model", "status"},
	)
	toolTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "agent_tool_total_test"},
		[]string{"tool", "status"},
	)
	c := &AgentCollector{
		runTotal:        runTotal,
		llmRequestTotal: llmRequest,
		toolTotal:       toolTotal,
	}

	c.IncRunTotal("ok")
	c.IncRunTotal("ok")
	c.IncRunTotal("failed")
	c.IncLLMRequest("gpt-4", "ok")
	c.IncLLMRequest("gpt-4", "ok")
	c.IncToolTotal("echo", "ok")

	expected := `
# HELP agent_run_total_test
# TYPE agent_run_total_test counter
agent_run_total_test{status="failed"} 1
agent_run_total_test{status="ok"} 2
`
	err := testutil.CollectAndCompare(runTotal, strings.NewReader(expected), "agent_run_total_test")
	assert.NoError(t, err)
}

func TestAgentCollector_HistogramAndRetryMetrics(t *testing.T) {
	turnDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: "agent_turn_duration_seconds_test", Buckets: prometheus.DefBuckets},
		[]string{"status"},
	)
	llmDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: "agent_llm_duration_seconds_test", Buckets: prometheus.DefBuckets},
		[]string{"model"},
	)
	llmRetry := prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "agent_llm_retry_total_test"},
		[]string{"model"},
	)
	compactTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "agent_compact_total_test"},
		[]string{"status"},
	)
	c := &AgentCollector{
		turnDuration:  turnDuration,
		llmDuration:   llmDuration,
		llmRetryTotal: llmRetry,
		compactTotal:  compactTotal,
	}

	c.ObserveTurnDuration("ok", 1.5)
	c.ObserveLLMDuration("claude", 2.0)
	c.IncLLMRetry("claude")
	c.IncCompact("ok")
	c.IncOverflowRetry("1")
	c.IncDoomLoop("run_terminal")
	c.IncSensorLint("ok")
	c.ObserveToolDuration("grep", 0.3)

	assert.NotNil(t, c.turnDuration)
	assert.NotNil(t, c.llmDuration)
}

func TestWorkflowCollector_HistogramMetrics(t *testing.T) {
	runDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: "workflow_run_duration_seconds_test", Buckets: prometheus.DefBuckets},
		[]string{"workflow", "status"},
	)
	stepDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: "workflow_step_duration_seconds_test", Buckets: prometheus.DefBuckets},
		[]string{"workflow", "step", "capability", "status"},
	)
	stepTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "workflow_step_total_test"},
		[]string{"workflow", "step", "status"},
	)
	c := &WorkflowCollector{
		runDuration:  runDuration,
		stepDuration: stepDuration,
		stepTotal:    stepTotal,
	}

	c.ObserveRunDuration("wf1", "done", 3.0)
	c.ObserveStepDuration("wf1", "step1", "capability", "done", 0.8)
	c.IncStepTotal("wf1", "step1", "done")
	c.IncStepTotal("wf1", "step1", "done")
	c.IncResume("wf1")

	expected := `
# HELP workflow_step_total_test
# TYPE workflow_step_total_test counter
workflow_step_total_test{status="done",step="step1",workflow="wf1"} 2
`
	err := testutil.CollectAndCompare(stepTotal, strings.NewReader(expected), "workflow_step_total_test")
	assert.NoError(t, err)
}

func TestEventCollector_MatchedAndDedup(t *testing.T) {
	matched := prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "event_matched_total_test"},
		[]string{"event_type", "pipeline"},
	)
	dedup := prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "event_dedup_total_test"},
		[]string{"event_type", "pipeline"},
	)
	lag := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: "event_lag_seconds_test", Buckets: prometheus.DefBuckets},
		[]string{"event_type"},
	)
	c := &EventCollector{matchedTotal: matched, dedupTotal: dedup, lagSeconds: lag}

	c.IncMatched("bookmark.created", "archive")
	c.IncMatched("bookmark.created", "archive")
	c.IncDedup("bookmark.created", "archive")
	c.ObserveLag("bookmark.created", 0.75)

	expected := `
# HELP event_matched_total_test
# TYPE event_matched_total_test counter
event_matched_total_test{event_type="bookmark.created",pipeline="archive"} 2
`
	err := testutil.CollectAndCompare(matched, strings.NewReader(expected), "event_matched_total_test")
	assert.NoError(t, err)
}

func TestCapabilityCollector_BulkheadMetrics(t *testing.T) {
	queued := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Name: "capability_bulkhead_queued_test"},
		[]string{"capability"},
	)
	active := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Name: "capability_bulkhead_active_test"},
		[]string{"capability"},
	)
	dropped := prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "capability_bulkhead_dropped_total_test"},
		[]string{"capability", "reason"},
	)
	waitDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: "capability_bulkhead_wait_seconds_test", Buckets: prometheus.DefBuckets},
		[]string{"capability"},
	)
	invokeDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: "capability_invoke_duration_seconds_test", Buckets: prometheus.DefBuckets},
		[]string{"capability", "operation"},
	)
	c := &CapabilityCollector{
		bulkheadQueued:       queued,
		bulkheadActive:       active,
		bulkheadDroppedTotal: dropped,
		bulkheadWaitDuration: waitDuration,
		invokeDuration:       invokeDuration,
	}

	c.IncBulkheadQueued("karakeep")
	c.IncBulkheadActive("karakeep")
	c.DecBulkheadQueued("karakeep")
	c.DecBulkheadActive("karakeep")
	c.IncBulkheadDropped("karakeep", "timeout")
	c.ObserveBulkheadWaitDuration("karakeep", 0.2)
	c.ObserveInvokeDuration("karakeep", "list", 0.4)

	assert.NotNil(t, c.bulkheadQueued)
	assert.NotNil(t, c.invokeDuration)
}

func TestNewCollectorsWithStats(t *testing.T) {
	st := stats.NewStats()

	tests := []struct {
		name string
		fn   func() any
	}{
		{name: "NewAgentCollector", fn: func() any { return NewAgentCollector(st) }},
		{name: "NewCapabilityCollector", fn: func() any { return NewCapabilityCollector(st) }},
		{name: "NewEventCollector", fn: func() any { return NewEventCollector(st) }},
		{name: "NewPipelineCollector", fn: func() any { return NewPipelineCollector(st) }},
		{name: "NewWorkflowCollector", fn: func() any { return NewWorkflowCollector(st) }},
		{name: "NewEventSourceCollector", fn: func() any { return NewEventSourceCollector(st) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.fn()
			assert.NotNil(t, c)
		})
	}
}
