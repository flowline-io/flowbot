package docker

import (
	"context"
	"testing"

	"github.com/docker/docker/api/types/volume"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/types"
)

func TestCreateVolume(t *testing.T) {
	t.Parallel()

	t.Run("create and remove volume via mounter", func(t *testing.T) {
		t.Parallel()
		skipIfNoDocker(t)
		vm, err := NewVolumeMounter()
		require.NoError(t, err)

		ctx := context.Background()
		mnt := &types.Mount{}
		err = vm.Mount(ctx, mnt)
		require.NoError(t, err)

		ls, err := vm.client.VolumeList(ctx, volume.ListOptions{})
		require.NoError(t, err)
		found := false
		for _, v := range ls.Volumes {
			if v.Name == mnt.Source {
				found = true
				break
			}
		}
		assert.True(t, found)

		err = vm.Unmount(ctx, mnt)
		require.NoError(t, err)

		ls, err = vm.client.VolumeList(ctx, volume.ListOptions{})
		require.NoError(t, err)

		for _, v := range ls.Volumes {
			assert.NotEqual(t, "testvol", v.Name)
		}
	})
}

func Test_createMountVolume(t *testing.T) {
	t.Parallel()

	t.Run("mount volume sets target and source", func(t *testing.T) {
		t.Parallel()
		skipIfNoDocker(t)
		m, err := NewVolumeMounter()
		require.NoError(t, err)

		mnt := &types.Mount{
			Type:   types.MountTypeVolume,
			Target: "/somevol",
		}

		err = m.Mount(context.Background(), mnt)
		require.NoError(t, err)
		defer func() {
			assert.NoError(t, m.Unmount(context.Background(), mnt))
		}()
		assert.Equal(t, "/somevol", mnt.Target)
		assert.NotEmpty(t, mnt.Source)
	})
}
