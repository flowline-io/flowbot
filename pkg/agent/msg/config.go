package msg

import "time"

const defaultMaxSteps = 50

// Config holds runtime options for an agent loop invocation.
type Config struct {
	MaxSteps      int
	ToolExecution ToolExecutionMode
	ModelName     string
	ChatModel     string
	ToolModel     string
	Temperature   float64
	MaxTokens     int
	// ThinkingLevel controls reasoning intensity for supported models (default/off/low/medium/high).
	ThinkingLevel       string
	TransformContext    TransformContextFn
	ConvertToLLM        ConvertToLLMFn
	PrepareNextTurn     PrepareNextTurnFn
	ShouldStopAfterTurn ShouldStopAfterTurnFn
	BeforeToolCall      BeforeToolCallFn
	AfterToolCall       AfterToolCallFn
	GetSteeringMessages GetMessagesFn
	GetFollowUpMessages GetMessagesFn
	SteeringMode        QueueMode
	FollowUpMode        QueueMode
	// LLMRetryMaxAttempts overrides default LLM retries when > 0.
	LLMRetryMaxAttempts int
	// LLMRetryInitialInterval overrides the first retry delay when > 0.
	LLMRetryInitialInterval time.Duration
	// LLMRetryMaxInterval caps the delay between retries when > 0.
	LLMRetryMaxInterval time.Duration
	// LLMRetryMultiplier controls delay growth when > 0.
	LLMRetryMultiplier float64
}

// WithDefaults fills zero values with package defaults.
func (c Config) WithDefaults() Config {
	if c.MaxSteps <= 0 {
		c.MaxSteps = defaultMaxSteps
	}
	if c.ToolExecution == "" {
		c.ToolExecution = ToolExecutionParallel
	}
	if c.SteeringMode == "" {
		c.SteeringMode = QueueAll
	}
	if c.FollowUpMode == "" {
		c.FollowUpMode = QueueAll
	}
	return c
}
