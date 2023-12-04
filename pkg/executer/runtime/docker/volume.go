package docker

import (
	"context"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/pkg/errors"
)

type VolumeMounter struct {
	client *client.Client
}

func NewVolumeMounter() (*VolumeMounter, error) {
	dc, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}
	return &VolumeMounter{client: dc}, nil
}

func (m *VolumeMounter) Mount(ctx context.Context, mn *types.Mount) error {
	name := utils.NewUUID()
	mn.Source = name
	v, err := m.client.VolumeCreate(ctx, volume.CreateOptions{Name: name})
	if err != nil {
		return err
	}
	flog.Debug("mount-point: %s, created volume %s", v.Mountpoint, v.Name)
	return nil
}

func (m *VolumeMounter) Unmount(ctx context.Context, mn *types.Mount) error {
	ls, err := m.client.VolumeList(ctx, volume.ListOptions{Filters: filters.NewArgs(filters.Arg("name", mn.Source))})
	if err != nil {
		return err
	}
	if len(ls.Volumes) == 0 {
		return errors.Errorf("unknown volume: %s", mn.Source)
	}
	if err := m.client.VolumeRemove(ctx, mn.Source, true); err != nil {
		return err
	}
	flog.Debug("removed volume %s", mn.Source)
	return nil
}
