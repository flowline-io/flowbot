package runtime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flowline-io/flowbot/pkg/types"
)

type fakeMounter struct{}

func (m *fakeMounter) Mount(ctx context.Context, mnt *types.Mount) error {
	return nil
}

func (m *fakeMounter) Unmount(ctx context.Context, mnt *types.Mount) error {
	return nil
}

func TestMultiVolumeMount(t *testing.T) {
	t.Parallel()

	t.Run("successful mount and unmount for registered volume type", func(t *testing.T) {
		t.Parallel()
		m := NewMultiMounter()
		m.RegisterMounter(types.MountTypeVolume, &fakeMounter{})
		ctx := context.Background()
		mnt := &types.Mount{
			Type:   types.MountTypeVolume,
			Target: "/mnt",
		}
		err := m.Mount(ctx, mnt)
		defer func() {
			err := m.Unmount(ctx, mnt)
			assert.NoError(t, err)
		}()
		assert.NoError(t, err)
	})
}

func TestMultiBadTypeMount(t *testing.T) {
	t.Parallel()

	t.Run("mount with unregistered type returns error", func(t *testing.T) {
		t.Parallel()
		m := NewMultiMounter()
		ctx := context.Background()
		mnt := &types.Mount{Type: "badone", Target: "/mnt"}
		err := m.Mount(ctx, mnt)
		assert.Error(t, err)
	})
}
