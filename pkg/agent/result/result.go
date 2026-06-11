// Package result provides a discriminated Result type for expected failures in the agent stack.
package result

// Result is the outcome of a fallible operation. Expected failures use IsOk() == false
// instead of panics or untyped errors at low-level boundaries.
type Result[T any, E error] struct {
	ok    bool
	value T
	err   E
}

// Ok constructs a successful Result.
func Ok[T any, E error](value T) Result[T, E] {
	return Result[T, E]{ok: true, value: value}
}

// Err constructs a failed Result.
func Err[T any, E error](err E) Result[T, E] {
	return Result[T, E]{ok: false, err: err}
}

// IsOk reports whether the operation succeeded.
func (r Result[T, E]) IsOk() bool {
	return r.ok
}

// Value returns the success value. Callers must check IsOk first.
func (r Result[T, E]) Value() T {
	return r.value
}

// ErrorValue returns the typed failure. Callers must check IsOk first.
func (r Result[T, E]) ErrorValue() E {
	return r.err
}

// ValueOrZero returns the value, typed error, and success flag for explicit branching.
func (r Result[T, E]) ValueOrZero() (T, E, bool) {
	return r.value, r.err, r.ok
}

// GetOrError returns the success value or the failure as a standard Go error.
// Intended for adapter boundaries such as Harness and Session public APIs.
func GetOrError[T any, E error](r Result[T, E]) (T, error) {
	if !r.ok {
		var zero T
		return zero, r.err
	}
	return r.value, nil
}
