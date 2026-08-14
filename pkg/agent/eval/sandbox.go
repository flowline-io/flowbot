package eval

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/flowline-io/flowbot/pkg/agent/env"
	agentsandbox "github.com/flowline-io/flowbot/pkg/agent/sandbox"
)

// Sandbox isolates per-case workspace state for eval trials.
type Sandbox interface {
	// Prepare creates the sandbox root for a case name under parent.
	Prepare(parent, caseName string) (string, error)
	// Reset clears the root and rewrites fixtures.
	Reset(root string, fixtures []WorkspaceFixture) error
	// Root returns the prepared root path (may be empty when unused).
	Root() string
}

// RuntimeSandbox can provide an execution environment bound to a workspace root.
type RuntimeSandbox interface {
	Sandbox
	ExecutionEnv(root string) env.ExecutionEnv
}

// WorkspaceSandbox is the default filesystem sandbox under {parent}/{caseName}.
type WorkspaceSandbox struct {
	root string
}

// Prepare implements Sandbox.
func (s *WorkspaceSandbox) Prepare(parent, caseName string) (string, error) {
	caseName = strings.TrimSpace(caseName)
	if caseName == "" {
		return "", fmt.Errorf("eval: sandbox case name required")
	}
	root := filepath.Join(parent, caseName)
	if err := os.MkdirAll(root, 0o750); err != nil {
		return "", err
	}
	s.root = root
	return root, nil
}

// Reset implements Sandbox.
func (s *WorkspaceSandbox) Reset(root string, fixtures []WorkspaceFixture) error {
	if strings.TrimSpace(root) == "" {
		root = s.root
	}
	if strings.TrimSpace(root) == "" {
		return nil
	}
	if err := os.RemoveAll(root); err != nil {
		return fmt.Errorf("eval: reset workspace: %w", err)
	}
	if err := os.MkdirAll(root, 0o750); err != nil {
		return err
	}
	s.root = root
	fx := make([]caseFixture, 0, len(fixtures))
	for _, f := range fixtures {
		fx = append(fx, caseFixture(f))
	}
	return writeFixtures(root, fx)
}

// Root implements Sandbox.
func (s *WorkspaceSandbox) Root() string {
	return s.root
}

// ExecutionEnv implements RuntimeSandbox.
func (*WorkspaceSandbox) ExecutionEnv(_ string) env.ExecutionEnv {
	return env.Default()
}

// NewWorkspaceSandbox returns an empty WorkspaceSandbox.
func NewWorkspaceSandbox() *WorkspaceSandbox {
	return &WorkspaceSandbox{}
}

// DockerSandboxConfig configures eval Docker sandbox behavior.
type DockerSandboxConfig struct {
	Image       string
	Network     string
	Memory      string
	ServerURL   string
	AccessToken string
}

const defaultEvalDockerImage = "ghcr.io/flowline-io/flowbot-agent-sandbox:latest"

// DockerSandbox provides eval workspace lifecycle and Docker-backed execution env.
type DockerSandbox struct {
	root string
	cfg  DockerSandboxConfig
}

// NewDockerSandbox creates a DockerSandbox.
func NewDockerSandbox(cfg DockerSandboxConfig) *DockerSandbox {
	if strings.TrimSpace(cfg.Image) == "" {
		cfg.Image = defaultEvalDockerImage
	}
	return &DockerSandbox{cfg: cfg}
}

// Prepare implements Sandbox.
func (s *DockerSandbox) Prepare(parent, caseName string) (string, error) {
	ws := &WorkspaceSandbox{}
	root, err := ws.Prepare(parent, caseName)
	if err != nil {
		return "", err
	}
	s.root = root
	return root, nil
}

// Reset implements Sandbox.
func (s *DockerSandbox) Reset(root string, fixtures []WorkspaceFixture) error {
	ws := &WorkspaceSandbox{root: s.root}
	if err := ws.Reset(root, fixtures); err != nil {
		return err
	}
	s.root = ws.root
	return nil
}

// Root implements Sandbox.
func (s *DockerSandbox) Root() string {
	return s.root
}

// ExecutionEnv implements RuntimeSandbox.
func (s *DockerSandbox) ExecutionEnv(root string) env.ExecutionEnv {
	if strings.TrimSpace(root) == "" {
		root = s.root
	}
	return agentsandbox.New(agentsandbox.Config{
		Image:       strings.TrimSpace(s.cfg.Image),
		Network:     strings.TrimSpace(s.cfg.Network),
		Memory:      strings.TrimSpace(s.cfg.Memory),
		Workspace:   root,
		ServerURL:   strings.TrimSpace(s.cfg.ServerURL),
		AccessToken: strings.TrimSpace(s.cfg.AccessToken),
	}, env.Default(), nil)
}
