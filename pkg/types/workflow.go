package types

import (
	"time"

	"github.com/cenkalti/backoff"
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

// RetryEnabled returns true if retries are configured with more than one attempt.
func (r *RetryConfig) RetryEnabled() bool {
	return r != nil && r.MaxAttempts > 1
}

// BuildBackOff constructs a backoff.BackOff from the retry configuration.
// Returns a StopBackOff if the config is nil.
func (r *RetryConfig) BuildBackOff() backoff.BackOff {
	if r == nil {
		return &backoff.StopBackOff{}
	}
	var bo backoff.BackOff
	switch r.Backoff {
	case BackoffFixed:
		bo = backoff.NewConstantBackOff(r.Delay)
	case BackoffLinear:
		bo = backoff.NewExponentialBackOff()
		ebo := bo.(*backoff.ExponentialBackOff)
		ebo.InitialInterval = r.Delay
		ebo.MaxInterval = r.MaxDelay
		ebo.Multiplier = 1.0
	case BackoffExponential, "":
		bo = backoff.NewExponentialBackOff()
		ebo := bo.(*backoff.ExponentialBackOff)
		ebo.InitialInterval = r.Delay
		ebo.MaxInterval = r.MaxDelay
		ebo.Multiplier = 2.0
	default:
		bo = backoff.NewExponentialBackOff()
		ebo := bo.(*backoff.ExponentialBackOff)
		ebo.InitialInterval = r.Delay
		ebo.MaxInterval = r.MaxDelay
		ebo.Multiplier = 2.0
	}
	if r.Jitter {
		if ebo, ok := bo.(*backoff.ExponentialBackOff); ok {
			ebo.RandomizationFactor = 0.5
		}
	}
	bo.Reset()
	// WithMaxRetries takes the number of retries (after the initial attempt),
	// so we subtract 1 from the total attempt count.
	return backoff.WithMaxRetries(bo, uint64(r.MaxAttempts-1))
}

type WorkflowMetadata struct {
	Name      string `json:"name" yaml:"name"`
	Describe  string `json:"describe" yaml:"describe"`
	Resumable bool   `json:"resumable" yaml:"resumable"`
	Triggers  []struct {
		Type string `json:"type" yaml:"type"`
		Rule KV     `json:"rule,omitempty" yaml:"rule"`
	} `json:"triggers" yaml:"triggers"`
	Pipeline []string       `json:"pipeline" yaml:"pipeline"`
	Tasks    []WorkflowTask `json:"tasks" yaml:"tasks"`
}

type WorkflowTask struct {
	ID       string       `json:"id" yaml:"id"`
	Action   string       `json:"action" yaml:"action"`
	Describe string       `json:"describe,omitempty" yaml:"describe"`
	Params   KV           `json:"params,omitempty" yaml:"params"`
	Vars     []string     `json:"vars,omitempty" yaml:"vars"`
	Conn     []string     `json:"conn,omitempty" yaml:"conn"`
	Retry    *RetryConfig `json:"retry,omitempty" yaml:"retry,omitempty"`
}
