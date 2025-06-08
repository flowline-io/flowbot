package docker

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestMountTmpfs(t *testing.T) {
	mounter := NewTmpfsMounter()
	ctx := context.Background()
	mnt := &types.Mount{
		Type:   types.MountTypeTmpfs,
		Target: "/target",
	}
	err := mounter.Mount(ctx, mnt)
	assert.NoError(t, err)
}

func TestMountTmpfsWithSource(t *testing.T) {
	mounter := NewTmpfsMounter()
	ctx := context.Background()
	mnt := &types.Mount{
		Type:   types.MountTypeTmpfs,
		Target: "/target",
		Source: "/source",
	}
	err := mounter.Mount(ctx, mnt)
	assert.Error(t, err)
}

func TestUnmountTmpfs(t *testing.T) {
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
}
