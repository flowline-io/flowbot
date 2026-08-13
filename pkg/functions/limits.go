package functions

import "time"

const (
	// MaxSourceBytes rejects function source above this size (256KiB).
	MaxSourceBytes = 256 << 10
	// MaxJSONBytes is the function JSON in/out limit (64KiB).
	MaxJSONBytes = 64 << 10
	// DefaultTimeout limits a single function invocation.
	DefaultTimeout = 60 * time.Second
	// DefaultMaxConcurrency is the global in-flight invoke limit.
	DefaultMaxConcurrency = 4
)
