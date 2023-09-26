package container

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"io"
	"os"
)

type Runtime struct {
	cli *client.Client
}

func NewRuntime() (*Runtime, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Runtime{
		cli: cli,
	}, nil
}

func (r *Runtime) ContainerList(ctx context.Context) ([]types.Container, error) {
	return r.cli.ContainerList(ctx, types.ContainerListOptions{
		All:  true,
		Size: true,
	})
}

func (r *Runtime) ContainerRemove(ctx context.Context, containerID string) error {
	return r.cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{
		Force: true,
	})
}

func (r *Runtime) ContainerLogs(ctx context.Context, containerID string) error {
	out, err := r.cli.ContainerLogs(ctx, containerID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Details:    true,
	})
	if err != nil {
		return err
	}
	_, err = stdcopy.StdCopy(os.Stdout, os.Stderr, out)
	return err
}

func (r *Runtime) ContainerWait(ctx context.Context, containerID string, condition container.WaitCondition) (<-chan container.WaitResponse, <-chan error) {
	return r.cli.ContainerWait(ctx, containerID, condition)
}

func (r *Runtime) ContainerStart(ctx context.Context, containerID string) error {
	return r.cli.ContainerStart(ctx, containerID, types.ContainerStartOptions{})
}

func (r *Runtime) ContainerCreate(ctx context.Context, name, image string, cmd []string) (container.CreateResponse, error) {
	return r.cli.ContainerCreate(ctx, &container.Config{
		Image:     image,
		Cmd:       cmd,
		Tty:       false,
		OpenStdin: true,
	}, nil, nil, nil, name)
}

func (r *Runtime) ImagePull(ctx context.Context, image string) error {
	reader, err := r.cli.ImagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()
	_, _ = io.Copy(os.Stdout, reader)
	return err
}
