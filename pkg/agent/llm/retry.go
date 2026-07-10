package llm

import (
	"errors"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/pkg/backoff"
	"github.com/flowline-io/flowbot/pkg/config"
)

// overflowPatterns mirrors ctxmgr overflow detection without importing ctxmgr
// (ctxmgr already depends on this package for Complete).
var overflowPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)prompt is too long`),
	regexp.MustCompile(`(?i)request_too_large`),
	regexp.MustCompile(`(?i)exceeds the context window`),
	regexp.MustCompile(`(?i)context[_ ]length[_ ]exceeded`),
	regexp.MustCompile(`(?i)too many tokens`),
	regexp.MustCompile(`(?i)token limit exceeded`),
	regexp.MustCompile(`(?i)maximum context length`),
}

var nonRetryableSubstrings = []string{
	"unauthorized",
	"invalid api key",
	"authentication",
	"permission denied",
	"forbidden",
}

var retryableSubstrings = []string{
	"429",
	"rate limit",
	"too many requests",
	"timeout",
	"timed out",
	"connection reset",
	"connection refused",
	"temporary failure",
	"service unavailable",
	"bad gateway",
	"gateway timeout",
	"503",
	"502",
	"504",
	"500",
}

// RetryConfig controls transient LLM call retries.
type RetryConfig struct {
	// MaxAttempts is the total number of execution attempts. Zero uses DefaultRetryConfig.
	MaxAttempts int
	// InitialInterval is the delay before the first retry. Zero uses DefaultRetryConfig.
	InitialInterval time.Duration
	// MaxInterval caps the delay between retries. Zero uses DefaultRetryConfig.
	MaxInterval time.Duration
	// Multiplier controls delay growth. Zero uses DefaultRetryConfig.
	Multiplier float64
	// OnRetry is called before each retry attempt. May be nil.
	OnRetry func(attempt int, delay time.Duration, err error)
}

// DefaultRetryConfig returns the default LLM retry policy.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:     3,
		InitialInterval: time.Second,
		MaxInterval:     30 * time.Second,
		Multiplier:      2.0,
	}
}

// RetryConfigFromChatAgent builds RetryConfig from chat agent settings.
func RetryConfigFromChatAgent(cfg config.LLMRetryConfig) RetryConfig {
	out := DefaultRetryConfig()
	if cfg.MaxAttempts > 0 {
		out.MaxAttempts = cfg.MaxAttempts
	}
	if cfg.InitialInterval > 0 {
		out.InitialInterval = cfg.InitialInterval
	}
	if cfg.MaxInterval > 0 {
		out.MaxInterval = cfg.MaxInterval
	}
	if cfg.Multiplier > 0 {
		out.Multiplier = cfg.Multiplier
	}
	return out
}

// IsRetryableLLMError reports whether an LLM error should be retried.
// Overflow, auth failures, and aborts are never retried.
func IsRetryableLLMError(err error) bool {
	if err == nil || errors.Is(err, ErrAborted) || errors.Is(err, ErrStreamStarted) {
		return false
	}
	errText := err.Error()
	if matchesOverflow(errText) || containsAny(strings.ToLower(errText), nonRetryableSubstrings) {
		return false
	}
	if containsAny(strings.ToLower(errText), retryableSubstrings) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr)
}

func matchesOverflow(errText string) bool {
	for _, re := range overflowPatterns {
		if re.MatchString(errText) {
			return true
		}
	}
	return false
}

func containsAny(text string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func (c RetryConfig) withDefaults() RetryConfig {
	def := DefaultRetryConfig()
	out := c
	if out.MaxAttempts <= 0 {
		out.MaxAttempts = def.MaxAttempts
	}
	if out.InitialInterval <= 0 {
		out.InitialInterval = def.InitialInterval
	}
	if out.MaxInterval <= 0 {
		out.MaxInterval = def.MaxInterval
	}
	if out.Multiplier <= 0 {
		out.Multiplier = def.Multiplier
	}
	return out
}

func (c RetryConfig) toBackoff() backoff.Config {
	cfg := c.withDefaults()
	return backoff.Config{
		MaxAttempts:     cfg.MaxAttempts,
		InitialInterval: cfg.InitialInterval,
		MaxInterval:     cfg.MaxInterval,
		Multiplier:      cfg.Multiplier,
		Jitter:          true,
		IsRetryable:     IsRetryableLLMError,
		OnRetry:         cfg.OnRetry,
	}
}

// ErrStreamStarted indicates a retryable transport error occurred after streaming
// deltas were already delivered to the client; callers must not retry.
var ErrStreamStarted = errors.New("agent llm: stream already started")

// streamStartedError wraps a cause that must not be retried because output was emitted.
type streamStartedError struct {
	cause error
}

func (e streamStartedError) Error() string {
	return ErrStreamStarted.Error() + ": " + e.cause.Error()
}

func (e streamStartedError) Unwrap() error { return e.cause }

func (streamStartedError) Is(target error) bool {
	return target == ErrStreamStarted
}
