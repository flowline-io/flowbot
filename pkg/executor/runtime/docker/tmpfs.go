package docker

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/types"
)

type TmpfsMounter struct {
}

func NewTmpfsMounter() *TmpfsMounter {
	return &TmpfsMounter{}
}

func (*TmpfsMounter) Mount(ctx context.Context, mnt *types.Mount) error {
	if mnt.Target == "" {
		return fmt.Errorf("tmpfs target is required")
	}
	if mnt.Source != "" {
		return fmt.Errorf("tmpfs source should be empty")
	}
	return nil
}

func (*TmpfsMounter) Unmount(ctx context.Context, mnt *types.Mount) error {
	return nil
}
