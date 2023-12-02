package docker

import (
	"context"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/pkg/errors"
)

type TmpfsMounter struct {
}

func NewTmpfsMounter() *TmpfsMounter {
	return &TmpfsMounter{}
}

func (m *TmpfsMounter) Mount(ctx context.Context, mnt *types.Mount) error {
	if mnt.Target == "" {
		return errors.Errorf("tmpfs target is required")
	}
	if mnt.Source != "" {
		return errors.Errorf("tmpfs source should be empty")
	}
	return nil
}

func (m *TmpfsMounter) Unmount(ctx context.Context, mnt *types.Mount) error {
	return nil
}
