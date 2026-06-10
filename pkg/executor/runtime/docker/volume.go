package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/utils"
)

// VolumeMounter manages Docker volume lifecycle: creation on Mount, removal on Unmount.
type VolumeMounter struct {
	client *client.Client
}

// NewVolumeMounter creates a VolumeMounter with its own Docker client.
func NewVolumeMounter() (*VolumeMounter, error) {
	dc, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &VolumeMounter{client: dc}, nil
}

// NewVolumeMounterWithClient creates a VolumeMounter using the provided Docker
// client, allowing the client to be shared with the Runtime to reduce resource usage.
func NewVolumeMounterWithClient(c *client.Client) *VolumeMounter {
	return &VolumeMounter{client: c}
}

// Mount creates a new Docker volume and sets its generated name on mn.Source.
func (m *VolumeMounter) Mount(ctx context.Context, mn *types.Mount) error {
	name := utils.NewUUID()
	mn.Source = name
	v, err := m.client.VolumeCreate(ctx, volume.CreateOptions{Name: name})
	if err != nil {
		return err
	}
	flog.Info("mount-point: %s, created volume %s", v.Mountpoint, v.Name)
	return nil
}

// Unmount removes the Docker volume identified by mn.Source.
func (m *VolumeMounter) Unmount(ctx context.Context, mn *types.Mount) error {
	ls, err := m.client.VolumeList(ctx, volume.ListOptions{Filters: filters.NewArgs(filters.Arg("name", mn.Source))})
	if err != nil {
		return err
	}
	if len(ls.Volumes) == 0 {
		return fmt.Errorf("unknown volume: %s", mn.Source)
	}
	if err := m.client.VolumeRemove(ctx, mn.Source, true); err != nil {
		return err
	}
	flog.Info("removed volume %s", mn.Source)
	return nil
}
