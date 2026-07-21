package types

import (
	"time"

	flowbackoff "github.com/flowline-io/flowbot/pkg/backoff"
)

// RetryConfig defines the retry strategy for a pipeline step or workflow task.
type RetryConfig struct {
	MaxAttempts int           `json:"max_attempts" yaml:"max_attempts"` // Total execution attempts; 0 or 1 means no retry.
	Delay       time.Duration `json:"delay" yaml:"delay"`
	Backoff     string        `json:"backoff" yaml:"backoff"` // fixed | linear | exponential
	MaxDelay    time.Duration `json:"max_delay" yaml:"max_delay"`
	Jitter      bool          `json:"jitter" yaml:"jitter"`
	RetryOn     []string      `json:"retry_on,omitempty" yaml:"retry_on,omitempty"`
}

// Backoff constants for RetryConfig.Backoff.
const (
	BackoffFixed       = "fixed"
	BackoffLinear      = "linear"
	BackoffExponential = "exponential"
)

// ToBackoffConfig converts RetryConfig to the unified backoff.Config.
func (r *RetryConfig) ToBackoffConfig() flowbackoff.Config {
	if r == nil {
		return flowbackoff.Config{MaxAttempts: 0}
	}
	multiplier := 2.0
	switch r.Backoff {
	case BackoffFixed, BackoffLinear:
		multiplier = 1.0
	}
	return flowbackoff.Config{
		MaxAttempts:     r.MaxAttempts,
		InitialInterval: r.Delay,
		MaxInterval:     r.MaxDelay,
		Multiplier:      multiplier,
		Jitter:          r.Jitter,
		RetryOn:         r.RetryOn,
	}
}

// WorkflowInputType enumerates supported workflow input value types.
const (
	WorkflowInputTypeString  = "string"
	WorkflowInputTypeNumber  = "number"
	WorkflowInputTypeBoolean = "boolean"
	WorkflowInputTypeJSON    = "json"
)

// WorkflowInputDef declares a top-level workflow input parameter.
type WorkflowInputDef struct {
	Name        string `json:"name" yaml:"name"`
	Type        string `json:"type" yaml:"type"` // string | number | boolean | json
	Required    bool   `json:"required,omitempty" yaml:"required,omitempty"`
	Default     any    `json:"default,omitempty" yaml:"default,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// WorkflowTriggerDef defines a single trigger for a workflow.
type WorkflowTriggerDef struct {
	Type    string `json:"type" yaml:"type"`
	Enabled bool   `json:"enabled" yaml:"enabled"`
	Rule    KV     `json:"rule,omitempty" yaml:"rule"`
}

// WorkflowMetadata is the canonical workflow definition used by YAML exchange and the runtime.
type WorkflowMetadata struct {
	Name           string               `json:"name" yaml:"name"`
	Describe       string               `json:"describe" yaml:"describe"`
	Enabled        bool                 `json:"enabled" yaml:"enabled"`
	Resumable      bool                 `json:"resumable" yaml:"resumable"`
	MaxConcurrency int                  `json:"max_concurrency" yaml:"max_concurrency"` // 0 or 1 = sequential; >1 enables DAG-based parallel execution
	Inputs         []WorkflowInputDef   `json:"inputs,omitempty" yaml:"inputs,omitempty"`
	Triggers       []WorkflowTriggerDef `json:"triggers" yaml:"triggers"`
	Pipeline       []string             `json:"pipeline" yaml:"pipeline"`
	Tasks          []WorkflowTask       `json:"tasks" yaml:"tasks"`
}

// WorkflowTask is a single step in a workflow DAG.
type WorkflowTask struct {
	ID       string       `json:"id" yaml:"id"`
	Action   string       `json:"action" yaml:"action"`
	Describe string       `json:"describe,omitempty" yaml:"describe"`
	Params   KV           `json:"params,omitempty" yaml:"params"`
	Vars     []string     `json:"vars,omitempty" yaml:"vars"`
	Conn     []string     `json:"conn,omitempty" yaml:"conn"`
	Retry    *RetryConfig `json:"retry,omitempty" yaml:"retry,omitempty"`
}
