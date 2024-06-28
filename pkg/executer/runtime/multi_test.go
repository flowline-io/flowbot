package runtime

import (
	"context"
	"testing"

	"github.com/flowline-io/flowbot/internal/types"
	"github.com/stretchr/testify/assert"
)

type fakeMounter struct{}

func (m *fakeMounter) Mount(ctx context.Context, mnt *types.Mount) error {
	return nil
}

func (m *fakeMounter) Unmount(ctx context.Context, mnt *types.Mount) error {
	return nil
}

func TestMultiVolumeMount(t *testing.T) {
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
}

func TestMultiBadTypeMount(t *testing.T) {
	m := NewMultiMounter()
	ctx := context.Background()
	mnt := &types.Mount{Type: "badone", Target: "/mnt"}
	err := m.Mount(ctx, mnt)
	assert.Error(t, err)
}
