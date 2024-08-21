package docker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
)

type BindMounter struct {
	cfg    BindConfig
	mounts map[string]string
	mu     sync.RWMutex
}

type BindConfig struct {
	Allowed bool
}

func NewBindMounter(cfg BindConfig) *BindMounter {
	return &BindMounter{
		cfg:    cfg,
		mounts: make(map[string]string),
	}
}

func (m *BindMounter) Mount(ctx context.Context, mnt *types.Mount) error {
	if !m.cfg.Allowed {
		return errors.New("bind mounts are not allowed")
	}
	m.mu.RLock()
	_, ok := m.mounts[mnt.Source]
	m.mu.RUnlock()
	if ok {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	// check if the source dir exists
	if _, err := os.Stat(mnt.Source); os.IsNotExist(err) {
		if err := os.MkdirAll(mnt.Source, 0707); err != nil {
			return fmt.Errorf("error creating mount directory: %s, %w", mnt.Source, err)
		}
		flog.Info("Created bind mount: %s", mnt.Source)
	} else if err != nil {
		return fmt.Errorf("error stat on directory: %s, %w", mnt.Source, err)
	}
	m.mounts[mnt.Source] = mnt.Source
	return nil
}

func (m *BindMounter) Unmount(ctx context.Context, mnt *types.Mount) error {
	return nil
}
