package eval

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Sandbox isolates per-case workspace state for eval trials.
// Docker implementations are deferred to a later phase; only WorkspaceSandbox ships now.
type Sandbox interface {
	// Prepare creates the sandbox root for a case name under parent.
	Prepare(parent, caseName string) (string, error)
	// Reset clears the root and rewrites fixtures.
	Reset(root string, fixtures []WorkspaceFixture) error
	// Root returns the prepared root path (may be empty when unused).
	Root() string
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
	if err := os.MkdirAll(root, 0o755); err != nil {
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
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	s.root = root
	fx := make([]caseFixture, 0, len(fixtures))
	for _, f := range fixtures {
		fx = append(fx, caseFixture{Path: f.Path, Content: f.Content})
	}
	return writeFixtures(root, fx)
}

// Root implements Sandbox.
func (s *WorkspaceSandbox) Root() string {
	return s.root
}

// NewWorkspaceSandbox returns an empty WorkspaceSandbox.
func NewWorkspaceSandbox() *WorkspaceSandbox {
	return &WorkspaceSandbox{}
}
