package msg

const defaultMaxSteps = 50

// Config holds runtime options for an agent loop invocation.
type Config struct {
	MaxSteps            int
	ToolExecution       ToolExecutionMode
	ModelName           string
	ChatModel           string
	ToolModel           string
	Temperature         float64
	MaxTokens           int
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
