package docker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

// BindMounter manages host-path bind mounts with an allow/deny policy.
type BindMounter struct {
	cfg    BindConfig
	mounts map[string]string
	mu     sync.Mutex
}

// BindConfig controls bind mount behavior.
type BindConfig struct {
	Allowed bool
}

// NewBindMounter creates a new BindMounter with the given configuration.
func NewBindMounter(cfg BindConfig) *BindMounter {
	return &BindMounter{
		cfg:    cfg,
		mounts: make(map[string]string),
	}
}

// Mount ensures the source directory exists and registers the bind mount.
// It returns an error if bind mounts are not allowed.
func (m *BindMounter) Mount(_ context.Context, mnt *types.Mount) error {
	if !m.cfg.Allowed {
		return errors.New("bind mounts are not allowed")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.mounts[mnt.Source]; ok {
		return nil
	}
	// Check if the source dir exists
	if _, err := os.Stat(mnt.Source); os.IsNotExist(err) {
		if err := os.MkdirAll(mnt.Source, 0o750); err != nil {
			return fmt.Errorf("error creating mount directory: %s, %w", mnt.Source, err)
		}
		flog.Info("Created bind mount: %s", mnt.Source)
	} else if err != nil {
		return fmt.Errorf("error stat on directory: %s, %w", mnt.Source, err)
	}
	m.mounts[mnt.Source] = mnt.Source
	return nil
}

// Unmount is a no-op for bind mounts since the host path persists after use.
func (*BindMounter) Unmount(_ context.Context, _ *types.Mount) error {
	return nil
}
