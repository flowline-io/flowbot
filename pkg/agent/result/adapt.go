package result

import "fmt"

// FileResultOrSessionError adapts a filesystem Result to a SessionError at storage boundaries.
func FileResultOrSessionError[T any](r Result[T, FileError], message string) (T, error) {
	if r.IsOk() {
		return r.Value(), nil
	}
	fileErr := r.ErrorValue()
	code := "storage"
	if fileErr.Code() == "not_found" {
		code = "not_found"
	}
	var zero T
	return zero, NewSessionError(code, fmt.Sprintf("%s: %s", message, fileErr.Message), fileErr)
}

// ToHarnessError wraps a subsystem failure as a HarnessError for public harness APIs.
func ToHarnessError(subsystem, message string, cause error) error {
	return NewHarnessError(subsystem, message, cause)
}
