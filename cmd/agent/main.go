// Command flowbot-agent is the local headless coding agent CLI.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/flowline-io/flowbot/cmd/agent/config"
	"github.com/flowline-io/flowbot/cmd/agent/run"
	"github.com/flowline-io/flowbot/pkg/agent/dcg"
	"github.com/flowline-io/flowbot/pkg/flog"
)

func main() {
	os.Exit(runMain(os.Args[1:]))
}

func runMain(args []string) int {
	fs := flag.NewFlagSet("flowbot-agent", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	printMode := fs.Bool("p", false, "print mode (non-interactive; required)")
	printLong := fs.Bool("print", false, "alias for -p")
	force := fs.Bool("force", false, "allow file writes and terminal commands")
	_ = fs.Bool("trust", false, "accepted for Cursor CLI compatibility (no-op in v1)")
	workspace := fs.String("workspace", "", "workspace root (default: cwd)")
	outputFormat := fs.String("output-format", "text", "output format (text only in v1)")
	cfgPath := fs.String("config", "agent.yaml", "path to agent.yaml")
	logLevel := fs.String("log-level", "info", "log level: debug, info, warn, error")
	timeout := fs.Duration("timeout", 30*time.Minute, "run timeout")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	flog.Init(flog.Config{Level: *logLevel})

	opts, err := parseCLI(cliInput{
		Print:        *printMode || *printLong,
		Force:        *force,
		Workspace:    *workspace,
		OutputFormat: *outputFormat,
		PromptArgs:   fs.Args(),
	})
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		flog.Error(fmt.Errorf("load config: %w", err))
		return 1
	}
	if err := cfg.Validate(); err != nil {
		flog.Error(fmt.Errorf("invalid config: %w", err))
		return 1
	}

	dcg.Init()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	result, err := run.Execute(ctx, run.Options{
		Config:    cfg,
		Workspace: opts.Workspace,
		Prompt:    opts.Prompt,
		Force:     opts.Force,
		Timeout:   *timeout,
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		flog.Error(fmt.Errorf("run failed: %w", err))
		if result.Text != "" {
			_, _ = fmt.Fprintln(os.Stdout, result.Text)
		}
		return 1
	}
	_, _ = fmt.Fprintln(os.Stdout, result.Text)
	return 0
}

type cliInput struct {
	Print        bool
	Force        bool
	Workspace    string
	OutputFormat string
	PromptArgs   []string
}

type parsedCLI struct {
	Force     bool
	Workspace string
	Prompt    string
}

func parseCLI(in cliInput) (parsedCLI, error) {
	if !in.Print {
		return parsedCLI{}, fmt.Errorf("headless mode requires -p / --print")
	}
	format := strings.ToLower(strings.TrimSpace(in.OutputFormat))
	if format == "" {
		format = "text"
	}
	if format != "text" {
		return parsedCLI{}, fmt.Errorf("unsupported --output-format %q (v1 supports text only)", in.OutputFormat)
	}
	prompt := strings.TrimSpace(strings.Join(in.PromptArgs, " "))
	if prompt == "" {
		return parsedCLI{}, fmt.Errorf("prompt is required")
	}
	ws := strings.TrimSpace(in.Workspace)
	if ws == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return parsedCLI{}, fmt.Errorf("workspace: %w", err)
		}
		ws = cwd
	}
	abs, err := filepath.Abs(ws)
	if err != nil {
		return parsedCLI{}, fmt.Errorf("workspace: %w", err)
	}
	return parsedCLI{
		Force:     in.Force,
		Workspace: abs,
		Prompt:    prompt,
	}, nil
}
