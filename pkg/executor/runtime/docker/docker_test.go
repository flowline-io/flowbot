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

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/flowline-io/flowbot/pkg/executor/runtime"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func TestParseCPUs(t *testing.T) {
	parsed, err := parseCPUs(&types.TaskLimits{CPUs: ".25"})
	assert.NoError(t, err)
	assert.Equal(t, int64(250000000), parsed)

	parsed, err = parseCPUs(&types.TaskLimits{CPUs: "1"})
	assert.NoError(t, err)
	assert.Equal(t, int64(1000000000), parsed)

	parsed, err = parseCPUs(&types.TaskLimits{CPUs: "0.5"})
	assert.NoError(t, err)
	assert.Equal(t, int64(500000000), parsed)
}

func TestPrintableReader(t *testing.T) {
	var s []byte
	for i := 0; i < 1000; i++ {
		s = append(s, 0)
	}
	s = append(s, []byte{104, 101, 108, 108, 111, 32, 119, 111, 114, 108, 100}...)
	pr := printableReader{reader: bytes.NewReader(s)}
	b, err := io.ReadAll(pr)
	assert.NoError(t, err)
	assert.Equal(t, "hello world", string(b))
}

type eofReader struct {
}

func (r eofReader) Read(p []byte) (int, error) {
	data := []byte{104, 101, 108, 108, 111, 32, 119, 111, 114, 108, 100}
	copy(p, data)
	return len(data), io.EOF
}

func TestPrintableReaderEOF(t *testing.T) {
	pr := printableReader{reader: eofReader{}}
	b, err := io.ReadAll(pr)
	assert.NoError(t, err)
	assert.Equal(t, "hello world", string(b))
}

func TestParseMemory(t *testing.T) {
	parsed, err := parseMemory(&types.TaskLimits{Memory: "1MB"})
	assert.NoError(t, err)
	assert.Equal(t, int64(1048576), parsed)

	parsed, err = parseMemory(&types.TaskLimits{Memory: "10MB"})
	assert.NoError(t, err)
	assert.Equal(t, int64(10485760), parsed)

	parsed, err = parseMemory(&types.TaskLimits{Memory: "500KB"})
	assert.NoError(t, err)
	assert.Equal(t, int64(512000), parsed)

	parsed, err = parseMemory(&types.TaskLimits{Memory: "1B"})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), parsed)
}

func TestNewDockerRuntime(t *testing.T) {
	rt, err := NewRuntime()
	assert.NoError(t, err)
	assert.NotNil(t, rt)
}

func TestRunTaskCMD(t *testing.T) {
	rt, err := NewRuntime()
	assert.NoError(t, err)
	assert.NotNil(t, rt)

	err = rt.Run(context.Background(), &types.Task{
		ID:    utils.NewUUID(),
		Image: "ubuntu:mantic",
		CMD:   []string{"ls"},
	})
	assert.NoError(t, err)
}

func TestRunTaskConcurrently(t *testing.T) {
	rt, err := NewRuntime()
	assert.NoError(t, err)
	assert.NotNil(t, rt)
	wg := sync.WaitGroup{}
	c := 10
	wg.Add(10)
	for i := 0; i < c; i++ {
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
}

func TestRunTaskWithTimeout(t *testing.T) {
	rt, err := NewRuntime()
	assert.NoError(t, err)
	assert.NotNil(t, rt)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err = rt.Run(ctx, &types.Task{
		ID:    utils.NewUUID(),
		Image: "ubuntu:mantic",
		CMD:   []string{"sleep", "10"},
	})
	assert.Error(t, err)
}

func TestRunTaskWithError(t *testing.T) {
	rt, err := NewRuntime()
	assert.NoError(t, err)
	assert.NotNil(t, rt)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err = rt.Run(ctx, &types.Task{
		ID:    utils.NewUUID(),
		Image: "ubuntu:mantic",
		Run:   "not_a_thing",
	})
	assert.Error(t, err)
}

func TestRunAndStopTask(t *testing.T) {
	rt, err := NewRuntime()
	assert.NoError(t, err)
	assert.NotNil(t, rt)
	t1 := &types.Task{
		ID:    utils.NewUUID(),
		Image: "ubuntu:mantic",
		CMD:   []string{"sleep", "10"},
	}
	go func() {
		err := rt.Run(context.Background(), t1)
		assert.Error(t, err)
	}()
	// give the task a chance to get started
	time.Sleep(time.Second)
	err = rt.Stop(context.Background(), t1)
	assert.NoError(t, err)
}

func TestHealthCheck(t *testing.T) {
	rt, err := NewRuntime()
	assert.NoError(t, err)
	assert.NotNil(t, rt)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	assert.NoError(t, rt.HealthCheck(ctx))
}

func TestHealthCheckFailed(t *testing.T) {
	rt, err := NewRuntime()
	assert.NoError(t, err)
	assert.NotNil(t, rt)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	assert.Error(t, rt.HealthCheck(ctx))
}

func TestRunTaskWithNetwork(t *testing.T) {
	rt, err := NewRuntime()
	assert.NoError(t, err)
	assert.NotNil(t, rt)
	err = rt.Run(context.Background(), &types.Task{
		ID:       utils.NewUUID(),
		Image:    "ubuntu:mantic",
		CMD:      []string{"ls"},
		Networks: []string{"default"},
	})
	assert.NoError(t, err)
	rt, err = NewRuntime()
	assert.NoError(t, err)
	assert.NotNil(t, rt)
	err = rt.Run(context.Background(), &types.Task{
		ID:       utils.NewUUID(),
		Image:    "ubuntu:mantic",
		CMD:      []string{"ls"},
		Networks: []string{"no-such-network"},
	})
	assert.Error(t, err)
}

func TestRunTaskWithVolume(t *testing.T) {
	rt, err := NewRuntime()
	assert.NoError(t, err)

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
	assert.NoError(t, err)
}

func TestRunTaskWithBind(t *testing.T) {
	mm := runtime.NewMultiMounter()
	vm, err := NewVolumeMounter()
	assert.NoError(t, err)
	mm.RegisterMounter("bind", NewBindMounter(BindConfig{Allowed: true}))
	mm.RegisterMounter("volume", vm)
	rt, err := NewRuntime(WithMounter(mm))
	assert.NoError(t, err)
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
	assert.NoError(t, err)
}

func TestRunTaskWithTempfs(t *testing.T) {
	rt, err := NewRuntime(
		WithMounter(NewTmpfsMounter()),
	)
	assert.NoError(t, err)

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
	assert.NoError(t, err)
}

func TestRunTaskWithCustomMounter(t *testing.T) {
	mounter := runtime.NewMultiMounter()
	vmounter, err := NewVolumeMounter()
	assert.NoError(t, err)
	mounter.RegisterMounter(types.MountTypeVolume, vmounter)
	rt, err := NewRuntime(WithMounter(mounter))
	assert.NoError(t, err)
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
	assert.NoError(t, err)
}

func TestRunTaskInitWorkdir(t *testing.T) {
	rt, err := NewRuntime()
	assert.NoError(t, err)
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
	assert.NoError(t, err)
	assert.Equal(t, "hello world", t1.Result)
}

func Test_imagePull(t *testing.T) {
	ctx := context.Background()

	rt, err := NewRuntime()
	assert.NoError(t, err)
	assert.NotNil(t, rt)

	images, err := rt.client.ImageList(ctx, image.ListOptions{
		Filters: filters.NewArgs(filters.Arg("reference", "busybox:*")),
	})
	assert.NoError(t, err)

	for _, img := range images {
		_, err = rt.client.ImageRemove(ctx, img.ID, image.RemoveOptions{Force: true})
		assert.NoError(t, err)
	}

	err = rt.imagePull(ctx, &types.Task{Image: "localhost:5001/no/suchthing"})
	assert.Error(t, err)

	wg := sync.WaitGroup{}
	wg.Add(3)

	for i := 0; i < 3; i++ {
		go func() {
			defer wg.Done()
			err := rt.imagePull(ctx, &types.Task{Image: "busybox:1.36"})
			assert.NoError(t, err)
		}()
	}
	wg.Wait()
}

func Test_imagePullPrivateRegistry(t *testing.T) {
	ctx := context.Background()

	rt, err := NewRuntime()
	assert.NoError(t, err)
	assert.NotNil(t, rt)

	r1, err := rt.client.ImagePull(ctx, "alpine:3.18.3", image.PullOptions{})
	assert.NoError(t, err)
	assert.NoError(t, r1.Close())

	images, err := rt.client.ImageList(ctx, image.ListOptions{
		Filters: filters.NewArgs(filters.Arg("reference", "alpine:3.18.3")),
	})
	assert.NoError(t, err)
	assert.Len(t, images, 1)

	err = rt.client.ImageTag(ctx, "alpine:3.18.3", "localhost:5001/flowbot/alpine:3.18.3")
	assert.NoError(t, err)

	r2, err := rt.client.ImagePush(ctx, "localhost:5001/flowbot/alpine:3.18.3", image.PushOptions{RegistryAuth: "noauth"})
	assert.NoError(t, err)
	assert.NoError(t, r2.Close())

	err = rt.imagePull(ctx, &types.Task{
		Image: "localhost:5001/flowbot/alpine:3.18.3",
		Registry: &types.Registry{
			Username: "username",
			Password: "password",
		},
	})

	assert.NoError(t, err)
}
