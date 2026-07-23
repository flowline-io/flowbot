package dcg

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/flowline-io/flowbot/pkg/flog"
)

// Checker evaluates a shell command string before execution.
type Checker interface {
	// Check returns a Decision for command, or an error for fail-closed failures.
	Check(ctx context.Context, command string) (Decision, error)
}

// commandRunner executes dcg and returns stdout, exit code, and spawn errors.
type commandRunner func(ctx context.Context, name string, args []string, env []string) (stdout string, exitCode int, err error)

// BinaryCheckerOptions configures NewBinaryChecker.
type BinaryCheckerOptions struct {
	// Path is the dcg executable path; empty uses LookPath("dcg").
	Path string
	// ConfigPath is passed as --config; required for production use.
	ConfigPath string
	// Runner overrides process execution (tests).
	Runner commandRunner
}

// BinaryChecker invokes the dcg CLI with --robot test.
type BinaryChecker struct {
	path       string
	configPath string
	runner     commandRunner
}

// NewBinaryChecker builds a Checker that shells out to dcg.
func NewBinaryChecker(opts BinaryCheckerOptions) *BinaryChecker {
	return &BinaryChecker{
		path:       strings.TrimSpace(opts.Path),
		configPath: strings.TrimSpace(opts.ConfigPath),
		runner:     opts.Runner,
	}
}

// Check runs dcg --robot test against command.
func (c *BinaryChecker) Check(ctx context.Context, command string) (Decision, error) {
	if c == nil {
		return Decision{}, fmt.Errorf("dcg: checker is nil")
	}
	command = strings.TrimSpace(command)
	if command == "" {
		return Decision{}, fmt.Errorf("dcg: command is required")
	}
	if c.configPath == "" {
		return Decision{}, fmt.Errorf("dcg: config path is required")
	}

	runner := c.runner
	cmdName := c.path
	if runner == nil {
		runner = defaultRunner
		var err error
		cmdName, err = resolveDCGPath(c.path)
		if err != nil {
			return Decision{}, err
		}
	} else if cmdName == "" {
		cmdName = "dcg"
	}

	// --robot forces JSON on stdout; do not place --format before the subcommand.
	args := []string{
		"--robot",
		"test",
		"--config", c.configPath,
		command,
	}
	env := StripBypassEnv(os.Environ())
	env = append(env, "DCG_CONFIG="+c.configPath)

	stdout, exitCode, err := runner(ctx, cmdName, args, env)
	if err != nil {
		if ctx.Err() != nil {
			flog.Warn("[dcg] check canceled command=%q: %v", TruncateCommandForLog(command), ctx.Err())
			return Decision{}, fmt.Errorf("dcg: check canceled: %w", ctx.Err())
		}
		flog.Warn("[dcg] check run failed command=%q: %v", TruncateCommandForLog(command), err)
		return Decision{}, fmt.Errorf("dcg: run failed: %w", err)
	}
	decision, err := parseRobotDecision(stdout, exitCode)
	if err != nil {
		flog.Warn("[dcg] check parse failed command=%q exit=%d: %v", TruncateCommandForLog(command), exitCode, err)
		return Decision{}, err
	}
	if decision.Allow {
		flog.Debug("[dcg] allow command=%q", TruncateCommandForLog(command))
	} else {
		flog.Warn("[dcg] deny command=%q rule=%s pack=%s reason=%s",
			TruncateCommandForLog(command), decision.RuleID, decision.PackID, decision.Reason)
	}
	return decision, nil
}

func resolveDCGPath(path string) (string, error) {
	if path == "" {
		resolved, err := exec.LookPath("dcg")
		if err != nil {
			return "", fmt.Errorf("dcg: binary not found on PATH: %w", err)
		}
		return resolved, nil
	}
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("dcg: binary not found: %w", err)
	}
	return path, nil
}

func defaultRunner(ctx context.Context, name string, args []string, env []string) (string, int, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = env
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = nil
	err := cmd.Run()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return stdout.String(), ee.ExitCode(), nil
		}
		return stdout.String(), 0, err
	}
	return stdout.String(), 0, nil
}

// StripBypassEnv returns env without DCG_BYPASS entries.
func StripBypassEnv(env []string) []string {
	out := make([]string, 0, len(env))
	for _, e := range env {
		key, _, _ := strings.Cut(e, "=")
		if strings.EqualFold(key, "DCG_BYPASS") {
			continue
		}
		out = append(out, e)
	}
	return out
}

// ErrorChecker always fails closed.
type ErrorChecker struct {
	// Err is returned from Check; nil uses a default message.
	Err error
}

// Check implements Checker.
func (c ErrorChecker) Check(context.Context, string) (Decision, error) {
	if c.Err != nil {
		return Decision{}, c.Err
	}
	return Decision{}, fmt.Errorf("dcg: unavailable")
}

// AllowAllChecker permits every command (tests).
type AllowAllChecker struct{}

// Check implements Checker.
func (AllowAllChecker) Check(context.Context, string) (Decision, error) {
	return Decision{Allow: true}, nil
}

// DenyChecker denies every command with Reason (tests).
type DenyChecker struct {
	// Reason is returned on denial.
	Reason string
}

// Check implements Checker.
func (c DenyChecker) Check(context.Context, string) (Decision, error) {
	reason := c.Reason
	if reason == "" {
		reason = "denied by test checker"
	}
	return Decision{Allow: false, Reason: reason}, nil
}

var (
	defaultMu      sync.RWMutex
	defaultChecker Checker = ErrorChecker{Err: fmt.Errorf("dcg: not initialized")}
)

// SetDefaultChecker installs the process-wide checker used when ChatHookDeps.DCG is nil.
func SetDefaultChecker(c Checker) {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	if c == nil {
		defaultChecker = ErrorChecker{Err: fmt.Errorf("dcg: not initialized")}
		return
	}
	defaultChecker = c
}

// DefaultChecker returns the process-wide checker.
func DefaultChecker() Checker {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return defaultChecker
}
