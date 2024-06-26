package docker

import (
	"context"
	"os"
	"sync"

	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"

	"github.com/pkg/errors"
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
			return errors.Wrapf(err, "error creating mount directory: %s", mnt.Source)
		}
		flog.Info("Created bind mount: %s", mnt.Source)
	} else if err != nil {
		return errors.Wrapf(err, "error stat on directory: %s", mnt.Source)
	}
	m.mounts[mnt.Source] = mnt.Source
	return nil
}

func (m *BindMounter) Unmount(ctx context.Context, mnt *types.Mount) error {
	return nil
}
