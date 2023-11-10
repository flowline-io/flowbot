package container

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"testing"
)

func TestRuntime(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "test container runtime",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			runtime, err := NewRuntime()
			if err != nil {
				t.Fatal(err)
			}

			// image pull
			err = runtime.ImagePull(ctx, "docker.io/library/alpine")
			if err != nil {
				t.Fatal(err)
			}

			// create
			resp, err := runtime.ContainerCreate(ctx, "test-alpine", "alpine", []string{"sh", "-c", "date"})
			if err != nil {
				t.Fatal(err)
			}

			// start
			err = runtime.ContainerStart(ctx, resp.ID)
			if err != nil {
				t.Fatal(err)
			}

			// wait
			statusCh, errCh := runtime.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
			select {
			case err = <-errCh:
				if err != nil {
					t.Fatal(err)
				}
			case <-statusCh:
			}

			// logs
			err = runtime.ContainerLogs(ctx, resp.ID)
			if err != nil {
				t.Fatal(err)
			}

			// remove
			err = runtime.ContainerRemove(ctx, resp.ID)
			if err != nil {
				t.Fatal(err)
			}

			// list
			list, err := runtime.ContainerList(ctx)
			if err != nil {
				t.Fatal(err)
			}
			t.Log(list)
		})
	}
}

func TestPlayground(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "test container playground",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			runtime, err := NewRuntime()
			if err != nil {
				t.Fatal(err)
			}

			// image pull
			err = runtime.ImagePull(ctx, "gcr.io/golang-org/playground-sandbox-gvisor:latest")
			if err != nil {
				t.Fatal(err)
			}

			// create
			resp, err := runtime.ContainerCreate(ctx,
				"test-playground-sandbox-gvisor",
				"gcr.io/golang-org/playground-sandbox-gvisor:latest",
				[]string{"/usr/local/bin/play-sandbox", "-mode", "contained"},
			)
			if err != nil {
				t.Fatal(err)
			}

			// start
			err = runtime.ContainerStart(ctx, resp.ID)
			if err != nil {
				t.Fatal(err)
			}

			// wait
			statusCh, errCh := runtime.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
			select {
			case err = <-errCh:
				if err != nil {
					t.Fatal(err)
				}
			case <-statusCh:
			}

			// logs
			err = runtime.ContainerLogs(ctx, resp.ID)
			if err != nil {
				t.Fatal(err)
			}

			// remove
			err = runtime.ContainerRemove(ctx, resp.ID)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRunCode(t *testing.T) {

	// create docker client
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Fatal(err)
	}

	// pull image
	image := "golang:latest"
	out, err := cli.ImagePull(context.Background(), image, types.ImagePullOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()

	// create container
	resp, err := cli.ContainerCreate(context.Background(), &container.Config{
		Image: image,
	}, nil, nil, nil, "")
	if err != nil {
		t.Fatal(err)
	}

	// upload code
	srcArchive, err := archive.TarWithOptions("./", &archive.TarOptions{
		IncludeFiles: []string{"container.go"},
	})
	if err != nil {
		t.Fatal(err)
	}
	err = cli.CopyToContainer(context.Background(), resp.ID, "/", srcArchive, types.CopyToContainerOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// begin exec
	execID, err := cli.ContainerExecCreate(context.Background(), resp.ID, types.ExecConfig{
		Cmd: []string{"golang", "run", "/container.go"},
	})
	if err != nil {
		t.Fatal(err)
	}
	err = cli.ContainerExecStart(context.Background(), execID.ID, types.ExecStartCheck{})
	if err != nil {
		t.Fatal(err)
	}

	// get exec
	_, _, err = cli.CopyFromContainer(context.Background(), resp.ID, "/")
	if err != nil {
		t.Fatal(err)
	}

	// remove
	err = cli.ContainerRemove(context.Background(), resp.ID, types.ContainerRemoveOptions{})
	if err != nil {
		t.Fatal(err)
	}
}
