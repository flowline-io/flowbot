package harness

import (
	"errors"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/agent/result"
)

// normalizeHarnessError wraps subsystem failures for public harness APIs.
func normalizeHarnessError(subsystem, message string, cause error) error {
	if cause == nil {
		return result.NewHarnessError(subsystem, message, nil)
	}
	var harnessErr result.HarnessError
	if errors.As(cause, &harnessErr) {
		return cause
	}
	return result.ToHarnessError(subsystem, message, cause)
}

// wrapPromptError adapts context manager and prompt preparation failures.
func wrapPromptError(err error) error {
	if err == nil {
		return nil
	}
	if result.IsCode(err, "nothing_to_compact") {
		return normalizeHarnessError("compaction", "context compaction required", err)
	}
	if result.IsCode(err, "summarization_failed") || result.IsCode(err, "aborted") {
		return normalizeHarnessError("compaction", err.Error(), err)
	}
	return fmt.Errorf("harness: %w", err)
}
