package runtime

import (
	"context"

	"github.com/flowline-io/flowbot/internal/types"
)

type Mounter interface {
	Mount(ctx context.Context, mnt *types.Mount) error
	Unmount(ctx context.Context, mnt *types.Mount) error
}
