package hooks

import "errors"

// ErrRunCancelled indicates a hook cancelled the run before the agent loop started.
var ErrRunCancelled = errors.New("agent hooks: run cancelled")
