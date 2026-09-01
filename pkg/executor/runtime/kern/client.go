package kern

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const defaultBinary = "kern"

// CommandRunner runs external commands; tests may replace it.
type CommandRunner func(ctx context.Context, name string, args ...string) *exec.Cmd

// Client invokes the kern CLI.
type Client struct {
	binary string
	run    CommandRunner
}

// NewClient creates a Client for the given config.
func NewClient(cfg Config) *Client {
	binary := strings.TrimSpace(cfg.Binary)
	if binary == "" {
		binary = defaultBinary
	}
	return &Client{
		binary: binary,
		run:    exec.CommandContext,
	}
}

// Binary returns the resolved kern executable path.
func (c *Client) Binary() string {
	return c.binary
}

// Doctor runs kern doctor.
func (c *Client) Doctor(ctx context.Context) error {
	cmd := c.run(ctx, c.binary, "doctor")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kern doctor: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Pull pulls an OCI image.
func (c *Client) Pull(ctx context.Context, image string) error {
	cmd := c.run(ctx, c.binary, "pull", image)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kern pull %s: %w: %s", image, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// BoxJob runs a one-shot kern box job and returns captured stdout/stderr and exit code.
func (c *Client) BoxJob(ctx context.Context, opts BoxOptions) (stdout, stderr string, exitCode int, err error) {
	args := opts.buildArgs(c.binary)
	cmd := c.run(ctx, args[0], args[1:]...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	runErr := cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()
	if runErr != nil {
		if ee, ok := runErr.(*exec.ExitError); ok {
			return stdout, stderr, ee.ExitCode(), nil
		}
		return stdout, stderr, -1, fmt.Errorf("kern box job: %w", runErr)
	}
	return stdout, stderr, 0, nil
}

// Stop stops a named box.
func (c *Client) Stop(ctx context.Context, name string) error {
	cmd := c.run(ctx, c.binary, "stop", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return fmt.Errorf("kern stop %s: %w", name, err)
		}
		return fmt.Errorf("kern stop %s: %w: %s", name, err, msg)
	}
	return nil
}

// BoxOptions configures a one-shot kern box job.
type BoxOptions struct {
	Name            string
	Image           string
	Command         []string
	Entrypoint      []string
	Env             []string
	Binds           []string
	Memory          string
	CPUs            string
	Network         string
	Workdir         string
	SecurityProfile string
	RequireLimits   bool
}

func (o BoxOptions) buildArgs(binary string) []string {
	args := []string{binary, "box", "job", "--name", o.Name, "--image", o.Image}
	if o.SecurityProfile != "" {
		args = append(args, "--security-profile", o.SecurityProfile)
	}
	if o.RequireLimits {
		args = append(args, "--require-limits")
	}
	if o.Memory != "" {
		args = append(args, "--memory", o.Memory)
	}
	if o.CPUs != "" {
		args = append(args, "--cpus", o.CPUs)
	}
	if o.Network == "host" {
		args = append(args, "--net", "host")
	}
	for _, bind := range o.Binds {
		args = append(args, "-v", bind)
	}
	for _, e := range o.Env {
		args = append(args, "-e", e)
	}
	if o.Workdir != "" {
		args = append(args, "--workdir", o.Workdir)
	}
	if len(o.Entrypoint) > 0 {
		args = append(args, "--entrypoint", strings.Join(o.Entrypoint, " "))
	}
	args = append(args, "--")
	args = append(args, o.Command...)
	return args
}

// BoxName sanitizes a task or session id into a kern box name.
func BoxName(id string) string {
	const prefix = "fb-"
	name := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '-'
		}
	}, strings.TrimSpace(id))
	if name == "" {
		name = "task"
	}
	const maxLen = 60
	if len(name) > maxLen {
		name = name[:maxLen]
	}
	return prefix + name
}

// ResolveBinary returns the configured binary or the default name.
func ResolveBinary(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return defaultBinary
	}
	return path
}

// LookPath reports whether the kern binary is available.
func LookPath(binary string) error {
	if strings.Contains(binary, string(os.PathSeparator)) {
		if _, err := os.Stat(binary); err != nil {
			return err
		}
		return nil
	}
	_, err := exec.LookPath(binary)
	return err
}
