// Package runner invokes local CLIs for CapGateway jobs.
package runner

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/types"
)

// Result is the outcome of one local CLI invocation.
type Result struct {
	Output   string
	ExitCode int
	Err      error
}

// Runner executes a gateway job against a local CLI.
type Runner interface {
	Run(ctx context.Context, job *types.GatewayJob, workspace string) Result
}

// Unsupported is a Runner that always fails (OpenCode placeholder).
type Unsupported struct {
	Name string
}

// Run implements Runner.
func (u Unsupported) Run(context.Context, *types.GatewayJob, string) Result {
	return Result{ExitCode: 1, Err: errUnsupported(u.Name)}
}

func errUnsupported(name string) error {
	if name == "" {
		name = "cli"
	}
	return &unsupportedError{name: name}
}

type unsupportedError struct{ name string }

func (e *unsupportedError) Error() string {
	return e.name + " runner is not enabled"
}
