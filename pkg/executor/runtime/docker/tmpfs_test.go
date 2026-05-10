package docker

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestMountTmpfs(t *testing.T) {
	t.Run("mount tmpfs with target only", func(t *testing.T) {
		mounter := NewTmpfsMounter()
		ctx := context.Background()
		mnt := &types.Mount{
			Type:   types.MountTypeTmpfs,
			Target: "/target",
		}
		err := mounter.Mount(ctx, mnt)
		assert.NoError(t, err)
	})
}

func TestMountTmpfsWithSource(t *testing.T) {
	t.Run("mount tmpfs with source returns error", func(t *testing.T) {
		mounter := NewTmpfsMounter()
		ctx := context.Background()
		mnt := &types.Mount{
			Type:   types.MountTypeTmpfs,
			Target: "/target",
			Source: "/source",
		}
		err := mounter.Mount(ctx, mnt)
		assert.Error(t, err)
	})
}

func TestUnmountTmpfs(t *testing.T) {
	t.Run("unmount tmpfs after mount", func(t *testing.T) {
		mounter := NewTmpfsMounter()
		ctx := context.Background()
		mnt := &types.Mount{
			Type:   types.MountTypeTmpfs,
			Target: "/target",
		}
		err := mounter.Mount(ctx, mnt)
		assert.NoError(t, err)
		err = mounter.Unmount(ctx, mnt)
		assert.NoError(t, err)
	})
}
