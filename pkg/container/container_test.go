package container

import (
	"context"
	"github.com/docker/docker/api/types/container"
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
