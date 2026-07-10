package metrics

import (
	"log"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/flowline-io/flowbot/pkg/stats"
)

// AgentCollector holds typed metrics for agent harness runs.
// When initialized with a nil stats, all methods are no-op.
type AgentCollector struct {
	runTotal           *prometheus.CounterVec
	turnDuration       *prometheus.HistogramVec
	llmRequestTotal    *prometheus.CounterVec
	llmRetryTotal      *prometheus.CounterVec
	llmDuration        *prometheus.HistogramVec
	toolTotal          *prometheus.CounterVec
	toolDuration       *prometheus.HistogramVec
	compactTotal       *prometheus.CounterVec
	overflowRetryTotal *prometheus.CounterVec
	doomLoopTotal      *prometheus.CounterVec
	sensorLintTotal    *prometheus.CounterVec
}

// NewAgentCollector creates an AgentCollector backed by stats.
// Returns a no-op collector when stats is nil or if registration fails.
func NewAgentCollector(st *stats.Stats) *AgentCollector {
	if st == nil {
		return &AgentCollector{}
	}
	var err error
	c := &AgentCollector{}
	c.runTotal, err = st.RegisterCounterVec("agent_run_total", "Agent runs by status", "status")
	if err != nil {
		log.Printf("[metrics] agent: failed to register run_total: %v", err)
		return &AgentCollector{}
	}
	c.turnDuration, err = st.RegisterHistogramVec("agent_turn_duration_seconds", "Agent turn duration", "status")
	if err != nil {
		log.Printf("[metrics] agent: failed to register turn_duration: %v", err)
		return &AgentCollector{}
	}
	c.llmRequestTotal, err = st.RegisterCounterVec("agent_llm_request_total", "LLM requests by model and status", "model", "status")
	if err != nil {
		log.Printf("[metrics] agent: failed to register llm_request_total: %v", err)
		return &AgentCollector{}
	}
	c.llmRetryTotal, err = st.RegisterCounterVec("agent_llm_retry_total", "LLM retries by model", "model")
	if err != nil {
		log.Printf("[metrics] agent: failed to register llm_retry_total: %v", err)
		return &AgentCollector{}
	}
	c.llmDuration, err = st.RegisterHistogramVec("agent_llm_duration_seconds", "LLM request duration by model", "model")
	if err != nil {
		log.Printf("[metrics] agent: failed to register llm_duration: %v", err)
		return &AgentCollector{}
	}
	c.toolTotal, err = st.RegisterCounterVec("agent_tool_total", "Tool executions by tool and status", "tool", "status")
	if err != nil {
		log.Printf("[metrics] agent: failed to register tool_total: %v", err)
		return &AgentCollector{}
	}
	c.toolDuration, err = st.RegisterHistogramVec("agent_tool_duration_seconds", "Tool execution duration by tool", "tool")
	if err != nil {
		log.Printf("[metrics] agent: failed to register tool_duration: %v", err)
		return &AgentCollector{}
	}
	c.compactTotal, err = st.RegisterCounterVec("agent_compact_total", "Context compaction events by status", "status")
	if err != nil {
		log.Printf("[metrics] agent: failed to register compact_total: %v", err)
		return &AgentCollector{}
	}
	c.overflowRetryTotal, err = st.RegisterCounterVec("agent_overflow_retry_total", "Overflow retries by level", "level")
	if err != nil {
		log.Printf("[metrics] agent: failed to register overflow_retry_total: %v", err)
		return &AgentCollector{}
	}
	c.doomLoopTotal, err = st.RegisterCounterVec("agent_doom_loop_total", "Doom loop detections by tool", "tool")
	if err != nil {
		log.Printf("[metrics] agent: failed to register doom_loop_total: %v", err)
		return &AgentCollector{}
	}
	c.sensorLintTotal, err = st.RegisterCounterVec("agent_sensor_lint_total", "Observation-only lint sensor events by status", "status")
	if err != nil {
		log.Printf("[metrics] agent: failed to register sensor_lint_total: %v", err)
		return &AgentCollector{}
	}
	return c
}

// IncRunTotal increments the agent run counter.
func (c *AgentCollector) IncRunTotal(status string) {
	if c.runTotal == nil {
		return
	}
	defer recoverLog("agent_run_total")
	c.runTotal.WithLabelValues(sanitizeLabel(status)).Inc()
}

// ObserveTurnDuration records a turn duration observation.
func (c *AgentCollector) ObserveTurnDuration(status string, seconds float64) {
	if c.turnDuration == nil {
		return
	}
	defer recoverLog("agent_turn_duration_seconds")
	c.turnDuration.WithLabelValues(sanitizeLabel(status)).Observe(seconds)
}

// IncLLMRequest increments the LLM request counter.
func (c *AgentCollector) IncLLMRequest(model, status string) {
	if c.llmRequestTotal == nil {
		return
	}
	defer recoverLog("agent_llm_request_total")
	c.llmRequestTotal.WithLabelValues(sanitizeLabel(model), sanitizeLabel(status)).Inc()
}

// IncLLMRetry increments the LLM retry counter.
func (c *AgentCollector) IncLLMRetry(model string) {
	if c.llmRetryTotal == nil {
		return
	}
	defer recoverLog("agent_llm_retry_total")
	c.llmRetryTotal.WithLabelValues(sanitizeLabel(model)).Inc()
}

// ObserveLLMDuration records an LLM request duration.
func (c *AgentCollector) ObserveLLMDuration(model string, seconds float64) {
	if c.llmDuration == nil {
		return
	}
	defer recoverLog("agent_llm_duration_seconds")
	c.llmDuration.WithLabelValues(sanitizeLabel(model)).Observe(seconds)
}

// IncToolTotal increments the tool execution counter.
func (c *AgentCollector) IncToolTotal(tool, status string) {
	if c.toolTotal == nil {
		return
	}
	defer recoverLog("agent_tool_total")
	c.toolTotal.WithLabelValues(sanitizeLabel(tool), sanitizeLabel(status)).Inc()
}

// ObserveToolDuration records a tool execution duration.
func (c *AgentCollector) ObserveToolDuration(tool string, seconds float64) {
	if c.toolDuration == nil {
		return
	}
	defer recoverLog("agent_tool_duration_seconds")
	c.toolDuration.WithLabelValues(sanitizeLabel(tool)).Observe(seconds)
}

// IncCompact increments the compaction counter.
func (c *AgentCollector) IncCompact(status string) {
	if c.compactTotal == nil {
		return
	}
	defer recoverLog("agent_compact_total")
	c.compactTotal.WithLabelValues(sanitizeLabel(status)).Inc()
}

// IncOverflowRetry increments the overflow retry counter for a degrade level.
func (c *AgentCollector) IncOverflowRetry(level string) {
	if c.overflowRetryTotal == nil {
		return
	}
	defer recoverLog("agent_overflow_retry_total")
	c.overflowRetryTotal.WithLabelValues(sanitizeLabel(level)).Inc()
}

// IncDoomLoop increments the doom loop counter.
func (c *AgentCollector) IncDoomLoop(tool string) {
	if c.doomLoopTotal == nil {
		return
	}
	defer recoverLog("agent_doom_loop_total")
	c.doomLoopTotal.WithLabelValues(sanitizeLabel(tool)).Inc()
}

// IncSensorLint increments the observation-only lint sensor counter.
func (c *AgentCollector) IncSensorLint(status string) {
	if c.sensorLintTotal == nil {
		return
	}
	defer recoverLog("agent_sensor_lint_total")
	c.sensorLintTotal.WithLabelValues(sanitizeLabel(status)).Inc()
}
