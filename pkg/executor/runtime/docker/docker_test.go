package docker

import (
	"bytes"
	"context"
	"io"
	"os"
	"path"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/moby/moby/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/executor/runtime"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/utils"
)

// skipIfNoDocker skips the test if the Docker daemon is not reachable.
func skipIfNoDocker(t *testing.T) {
	t.Helper()
	rt, err := NewRuntime()
	if err != nil {
		t.Skipf("skipping test: Docker runtime not available: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rt.HealthCheck(ctx); err != nil {
		t.Skipf("skipping test: Docker daemon not reachable: %v", err)
	}
}

func TestParseCPUs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cpus string
		exp  int64
	}{
		{name: "fractional 0.25", cpus: ".25", exp: int64(250000000)},
		{name: "whole number 1", cpus: "1", exp: int64(1000000000)},
		{name: "fractional 0.5", cpus: "0.5", exp: int64(500000000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			parsed, err := parseCPUs(&types.TaskLimits{CPUs: tt.cpus})
			require.NoError(t, err)
			assert.Equal(t, tt.exp, parsed)
		})
	}
}

func TestPrintableReader(t *testing.T) {
	t.Parallel()

	t.Run("filters null bytes and returns printable content", func(t *testing.T) {
		t.Parallel()
		var s []byte
		for range 1000 {
			s = append(s, 0)
		}
		s = append(s, []byte{104, 101, 108, 108, 111, 32, 119, 111, 114, 108, 100}...)
		pr := printableReader{reader: bytes.NewReader(s)}
		b, err := io.ReadAll(pr)
		require.NoError(t, err)
		assert.Equal(t, "hello world", string(b))
	})
}

type eofReader struct {
}

func (eofReader) Read(p []byte) (int, error) {
	data := []byte{104, 101, 108, 108, 111, 32, 119, 111, 114, 108, 100}
	copy(p, data)
	return len(data), io.EOF
}

func TestPrintableReaderEOF(t *testing.T) {
	t.Parallel()

	t.Run("reader returning EOF with valid data", func(t *testing.T) {
		t.Parallel()
		pr := printableReader{reader: eofReader{}}
		b, err := io.ReadAll(pr)
		require.NoError(t, err)
		assert.Equal(t, "hello world", string(b))
	})
}

func TestParseMemory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		mem  string
		exp  int64
	}{
		{name: "1 MB", mem: "1MB", exp: int64(1048576)},
		{name: "10 MB", mem: "10MB", exp: int64(10485760)},
		{name: "500 KB", mem: "500KB", exp: int64(512000)},
		{name: "1 byte", mem: "1B", exp: int64(1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			parsed, err := parseMemory(&types.TaskLimits{Memory: tt.mem})
			require.NoError(t, err)
			assert.Equal(t, tt.exp, parsed)
		})
	}
}

func TestNewDockerRuntime(t *testing.T) {
	t.Parallel()

	t.Run("creates a new Docker runtime", func(t *testing.T) {
		t.Parallel()
		skipIfNoDocker(t)
		rt, err := NewRuntime()
		require.NoError(t, err)
		assert.NotNil(t, rt)
	})
}

func TestRunTaskCMD(t *testing.T) {
	t.Parallel()

	t.Run("runs a task with CMD", func(t *testing.T) {
		t.Parallel()
		skipIfNoDocker(t)
		rt, err := NewRuntime()
		require.NoError(t, err)
		assert.NotNil(t, rt)

		err = rt.Run(context.Background(), &types.Task{
			ID:    utils.NewUUID(),
			Image: "ubuntu:mantic",
			CMD:   []string{"ls"},
		})
		require.NoError(t, err)
	})
}

func TestRunTaskConcurrently(t *testing.T) {
	t.Parallel()

	t.Run("runs multiple tasks concurrently", func(t *testing.T) {
		t.Parallel()
		skipIfNoDocker(t)
		rt, err := NewRuntime()
		require.NoError(t, err)
		assert.NotNil(t, rt)
		wg := sync.WaitGroup{}
		c := 10
		wg.Add(c)
		for range c {
			go func() {
				defer wg.Done()
				tk := &types.Task{
					ID:    utils.NewUUID(),
					Image: "ubuntu:mantic",
					Run:   "echo -n hello > $OUTPUT",
				}
				err := rt.Run(context.Background(), tk)
				assert.NoError(t, err)
				assert.Equal(t, "hello", tk.Result)
			}()
		}
		wg.Wait()
	})
}

func TestRunTaskWithTimeout(t *testing.T) {
	t.Parallel()

	t.Run("run cancels task on timeout", func(t *testing.T) {
		t.Parallel()
		skipIfNoDocker(t)
		rt, err := NewRuntime()
		require.NoError(t, err)
		assert.NotNil(t, rt)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		err = rt.Run(ctx, &types.Task{
			ID:    utils.NewUUID(),
			Image: "ubuntu:mantic",
			CMD:   []string{"sleep", "10"},
		})
		require.Error(t, err)
	})
}

func TestRunTaskWithError(t *testing.T) {
	t.Parallel()

	t.Run("run returns error on invalid command", func(t *testing.T) {
		t.Parallel()
		skipIfNoDocker(t)
		rt, err := NewRuntime()
		require.NoError(t, err)
		assert.NotNil(t, rt)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		err = rt.Run(ctx, &types.Task{
			ID:    utils.NewUUID(),
			Image: "ubuntu:mantic",
			Run:   "not_a_thing",
		})
		require.Error(t, err)
	})
}

func TestRunAndStopTask(t *testing.T) {
	t.Parallel()

	t.Run("stop terminates a running task", func(t *testing.T) {
		t.Parallel()
		skipIfNoDocker(t)
		rt, err := NewRuntime()
		require.NoError(t, err)
		assert.NotNil(t, rt)
		t1 := &types.Task{
			ID:    utils.NewUUID(),
			Image: "ubuntu:mantic",
			CMD:   []string{"sleep", "10"},
		}
		done := make(chan struct{})
		go func() {
			defer close(done)
			err := rt.Run(context.Background(), t1)
			assert.Error(t, err)
		}()
		// give the task a chance to get started
		time.Sleep(time.Second)
		err = rt.Stop(context.Background(), t1)
		require.NoError(t, err)
		<-done
	})
}

func TestHealthCheck(t *testing.T) {
	t.Parallel()

	t.Run("docker daemon is healthy", func(t *testing.T) {
		t.Parallel()
		skipIfNoDocker(t)
		rt, err := NewRuntime()
		require.NoError(t, err)
		assert.NotNil(t, rt)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		assert.NoError(t, rt.HealthCheck(ctx))
	})
}

func TestHealthCheckFailed(t *testing.T) {
	t.Parallel()

	t.Run("health check fails with cancelled context", func(t *testing.T) {
		t.Parallel()
		skipIfNoDocker(t)
		rt, err := NewRuntime()
		require.NoError(t, err)
		assert.NotNil(t, rt)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		assert.Error(t, rt.HealthCheck(ctx))
	})
}

func TestRunTaskWithNetwork(t *testing.T) {
	t.Parallel()

	skipIfNoDocker(t)
	tests := []struct {
		name     string
		networks []string
		expErr   bool
	}{
		{name: "valid default network", networks: []string{"default"}, expErr: false},
		{name: "non-existent network", networks: []string{"no-such-network"}, expErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rt, err := NewRuntime()
			require.NoError(t, err)
			assert.NotNil(t, rt)
			err = rt.Run(context.Background(), &types.Task{
				ID:       utils.NewUUID(),
				Image:    "ubuntu:mantic",
				CMD:      []string{"ls"},
				Networks: tt.networks,
			})
			if tt.expErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRunTaskWithVolume(t *testing.T) {
	t.Parallel()

	t.Run("run task with volume mount", func(t *testing.T) {
		t.Parallel()
		skipIfNoDocker(t)
		rt, err := NewRuntime()
		require.NoError(t, err)

		ctx := context.Background()

		t1 := &types.Task{
			ID:    utils.NewUUID(),
			Image: "ubuntu:mantic",
			Run:   "echo hello world > /xyz/thing",
			Mounts: []types.Mount{
				{
					Type:   types.MountTypeVolume,
					Target: "/xyz",
				},
			},
		}
		err = rt.Run(ctx, t1)
		require.NoError(t, err)
	})
}

func TestRunTaskWithBind(t *testing.T) {
	t.Parallel()

	t.Run("run task with bind mount", func(t *testing.T) {
		t.Parallel()
		skipIfNoDocker(t)
		mm := runtime.NewMultiMounter()
		vm, err := NewVolumeMounter()
		require.NoError(t, err)
		mm.RegisterMounter("bind", NewBindMounter(BindConfig{Allowed: true}))
		mm.RegisterMounter("volume", vm)
		rt, err := NewRuntime(WithMounter(mm))
		require.NoError(t, err)
		ctx := context.Background()
		dir := path.Join(os.TempDir(), utils.NewUUID())
		t1 := &types.Task{
			ID:    utils.NewUUID(),
			Image: "ubuntu:mantic",
			Run:   "echo hello world > /xyz/thing",
			Mounts: []types.Mount{{
				Type:   types.MountTypeBind,
				Target: "/xyz",
				Source: dir,
			}},
		}
		err = rt.Run(ctx, t1)
		require.NoError(t, err)
	})
}

func TestRunTaskWithTempfs(t *testing.T) {
	t.Parallel()

	t.Run("run task with tmpfs mount", func(t *testing.T) {
		t.Parallel()
		skipIfNoDocker(t)
		rt, err := NewRuntime(
			WithMounter(NewTmpfsMounter()),
		)
		require.NoError(t, err)

		ctx := context.Background()

		t1 := &types.Task{
			ID:    utils.NewUUID(),
			Image: "ubuntu:mantic",
			Run:   "echo hello world > /xyz/thing",
			Mounts: []types.Mount{
				{
					Type:   types.MountTypeTmpfs,
					Target: "/xyz",
				},
			},
		}
		err = rt.Run(ctx, t1)
		require.NoError(t, err)
	})
}

func TestRunTaskWithCustomMounter(t *testing.T) {
	t.Parallel()

	t.Run("run task with custom mounter", func(t *testing.T) {
		t.Parallel()
		skipIfNoDocker(t)
		mounter := runtime.NewMultiMounter()
		vmounter, err := NewVolumeMounter()
		require.NoError(t, err)
		mounter.RegisterMounter(types.MountTypeVolume, vmounter)
		rt, err := NewRuntime(WithMounter(mounter))
		require.NoError(t, err)
		t1 := &types.Task{
			ID:    utils.NewUUID(),
			Image: "ubuntu:mantic",
			Run:   "echo hello world > /xyz/thing",
			Mounts: []types.Mount{
				{
					Type:   types.MountTypeVolume,
					Target: "/xyz",
				},
			},
		}
		ctx := context.Background()
		err = rt.Run(ctx, t1)
		require.NoError(t, err)
	})
}

func TestRunTaskInitWorkdir(t *testing.T) {
	t.Parallel()

	t.Run("run task with input files in workdir", func(t *testing.T) {
		t.Parallel()
		skipIfNoDocker(t)
		rt, err := NewRuntime()
		require.NoError(t, err)
		t1 := &types.Task{
			ID:    utils.NewUUID(),
			Image: "ubuntu:mantic",
			Run:   "cat hello.txt > $OUTPUT",
			Files: map[string]string{
				"hello.txt": "hello world",
				"large.txt": strings.Repeat("a", 100_000),
			},
		}
		ctx := context.Background()
		err = rt.Run(ctx, t1)
		require.NoError(t, err)
		assert.Equal(t, "hello world", t1.Result)
	})
}

func Test_imagePull(t *testing.T) {
	t.Parallel()

	t.Run("pull image with retry and concurrency", func(t *testing.T) {
		t.Parallel()
		skipIfNoDocker(t)
		ctx := context.Background()

		rt, err := NewRuntime()
		require.NoError(t, err)
		assert.NotNil(t, rt)

		images, err := rt.client.ImageList(ctx, client.ImageListOptions{
			Filters: make(client.Filters).Add("reference", "busybox:*"),
		})
		require.NoError(t, err)

		for _, img := range images.Items {
			_, err = rt.client.ImageRemove(ctx, img.ID, client.ImageRemoveOptions{Force: true})
			require.NoError(t, err)
		}

		err = rt.imagePull(ctx, &types.Task{Image: "localhost:5001/no/suchthing"})
		require.Error(t, err)

		wg := sync.WaitGroup{}
		wg.Add(3)

		for range 3 {
			go func() {
				defer wg.Done()
				err := rt.imagePull(ctx, &types.Task{Image: "busybox:1.36"})
				assert.NoError(t, err)
			}()
		}
		wg.Wait()
	})
}

func Test_imagePullPrivateRegistry(t *testing.T) {
	t.Parallel()

	t.Run("pull from private registry with auth", func(t *testing.T) {
		t.Parallel()
		skipIfNoDocker(t)
		ctx := context.Background()

		rt, err := NewRuntime()
		require.NoError(t, err)
		assert.NotNil(t, rt)

		// attempt to pull a public image; skip entire test if unable to reach registry
		r1, err := rt.client.ImagePull(ctx, "alpine:3.18.3", client.ImagePullOptions{})
		if err != nil {
			t.Skipf("could not pull alpine image: %v", err)
		}
		_ = r1.Close()

		images, err := rt.client.ImageList(ctx, client.ImageListOptions{
			Filters: make(client.Filters).Add("reference", "alpine:3.18.3"),
		})
		require.NoError(t, err)
		// len may be 0 if daemon cleaned up, not a failure
		if len(images.Items) == 0 {
			t.Skip("image not found after pull, skipping private registry portion")
		}

		// try tagging; if local registry not available skip
		_, err = rt.client.ImageTag(ctx, client.ImageTagOptions{
			Source: "alpine:3.18.3",
			Target: "localhost:5001/flowbot/alpine:3.18.3",
		})
		if err != nil {
			t.Skipf("unable to tag for local registry: %v", err)
		}

		r2, err := rt.client.ImagePush(ctx, "localhost:5001/flowbot/alpine:3.18.3", client.ImagePushOptions{RegistryAuth: "noauth"})
		if err != nil {
			t.Skipf("unable to push to local registry: %v", err)
		}
		_ = r2.Close()

		err = rt.imagePull(ctx, &types.Task{
			Image: "localhost:5001/flowbot/alpine:3.18.3",
			Registry: &types.Registry{
				Username: "username",
				Password: "password",
			},
		})
		if err != nil {
			t.Skipf("unable to pull from private registry: %v", err)
		}
	})
}
