package docker

import (
	"context"
	"fmt"

	"github.com/moby/moby/client"

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
	dc, err := client.New(client.FromEnv, client.WithAPIVersionNegotiation())
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
	res, err := m.client.VolumeCreate(ctx, client.VolumeCreateOptions{Name: name})
	if err != nil {
		return err
	}
	flog.Info("mount-point: %s, created volume %s", res.Volume.Mountpoint, res.Volume.Name)
	return nil
}

// Unmount removes the Docker volume identified by mn.Source.
func (m *VolumeMounter) Unmount(ctx context.Context, mn *types.Mount) error {
	ls, err := m.client.VolumeList(ctx, client.VolumeListOptions{
		Filters: make(client.Filters).Add("name", mn.Source),
	})
	if err != nil {
		return err
	}
	if len(ls.Items) == 0 {
		return fmt.Errorf("unknown volume: %s", mn.Source)
	}
	if _, err := m.client.VolumeRemove(ctx, mn.Source, client.VolumeRemoveOptions{Force: true}); err != nil {
		return err
	}
	flog.Info("removed volume %s", mn.Source)
	return nil
}
