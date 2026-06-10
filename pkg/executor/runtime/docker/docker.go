package docker

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/bytedance/sonic"
	cliopts "github.com/docker/cli/opts"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	regtypes "github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/docker/go-units"

	"github.com/flowline-io/flowbot/pkg/executor/runtime"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/flowline-io/flowbot/pkg/utils/syncx"
)

type Runtime struct {
	client     *client.Client
	tasks      *syncx.Map[string, string]
	cancels    *syncx.Map[string, context.CancelFunc]
	images     *syncx.Map[string, bool]
	pullMu     *syncx.Map[string, *sync.Mutex]
	pullq      chan *pullRequest
	pullerDone chan struct{}
	mounter    runtime.Mounter
	config     string
}

type printableReader struct {
	reader io.Reader
}

type pullRequest struct {
	image    string
	registry registry
	done     chan error
}

type registry struct {
	username string
	password string
}

type Option = func(rt *Runtime)

func WithMounter(mounter runtime.Mounter) Option {
	return func(rt *Runtime) {
		rt.mounter = mounter
	}
}

// WithConfig sets the docker config file path for registry authentication.
func WithConfig(config string) Option {
	return func(rt *Runtime) {
		rt.config = config
	}
}

// WithClient sets a pre-existing Docker client, avoiding the creation of a
// duplicate client when sharing one across components (e.g. VolumeMounter).
func WithClient(c *client.Client) Option {
	return func(rt *Runtime) {
		rt.client = c
	}
}

func NewRuntime(opts ...Option) (*Runtime, error) {
	rt := &Runtime{
		tasks:      new(syncx.Map[string, string]),
		cancels:    new(syncx.Map[string, context.CancelFunc]),
		images:     new(syncx.Map[string, bool]),
		pullMu:     new(syncx.Map[string, *sync.Mutex]),
		pullq:      make(chan *pullRequest, 1),
		pullerDone: make(chan struct{}),
	}
	for _, o := range opts {
		o(rt)
	}
	// Create Docker client if not provided via WithClient.
	if rt.client == nil {
		dc, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			return nil, err
		}
		rt.client = dc
	}

	// setup a default mounter
	if rt.mounter == nil {
		vmounter, err := NewVolumeMounter()
		if err != nil {
			return nil, err
		}
		rt.mounter = vmounter
	}
	go rt.puller()
	return rt, nil
}

func (d *Runtime) Run(ctx context.Context, t *types.Task) error {
	// prepare mounts
	for i, mnt := range t.Mounts {
		err := d.mounter.Mount(ctx, &mnt)
		if err != nil {
			return err
		}
		defer func(m types.Mount) { //revive:disable
			uctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			defer cancel()
			if err := d.mounter.Unmount(uctx, &m); err != nil {
				flog.Error(fmt.Errorf("error unmounting mount: %s, %w", m, err))
			}
		}(mnt)
		t.Mounts[i] = mnt
	}
	// execute pre-tasks
	for _, pre := range t.Pre {
		pre.ID = utils.NewUUID()
		pre.Mounts = t.Mounts
		pre.Networks = t.Networks
		pre.Limits = t.Limits
		if err := d.doRun(ctx, pre); err != nil {
			return err
		}
	}
	// run the actual task
	if err := d.doRun(ctx, t); err != nil {
		return err
	}
	// execute post tasks
	for _, post := range t.Post {
		post.ID = utils.NewUUID()
		post.Mounts = t.Mounts
		post.Networks = t.Networks
		post.Limits = t.Limits
		if err := d.doRun(ctx, post); err != nil {
			return err
		}
	}
	return nil
}

func (d *Runtime) doRun(ctx context.Context, t *types.Task) error {
	if t.ID == "" {
		return errors.New("task id is required")
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	d.cancels.Set(t.ID, cancel)
	defer d.cancels.Delete(t.ID)

	if err := d.imagePull(ctx, t); err != nil {
		return fmt.Errorf("error pulling image: %s, %w", t.Image, err)
	}

	env := buildEnvVars(t)

	mounts, err := buildContainerMounts(t)
	if err != nil {
		return err
	}

	workdir := &types.Mount{Type: types.MountTypeVolume, Target: "/flowbot"}
	if err := d.mounter.Mount(ctx, workdir); err != nil {
		return err
	}
	defer func() {
		uctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		if err := d.mounter.Unmount(uctx, workdir); err != nil {
			flog.Error(fmt.Errorf("error unmounting workdir, %w", err))
		}
	}()
	mounts = append(mounts, mount.Mount{
		Type:   mount.TypeVolume,
		Source: workdir.Source,
		Target: workdir.Target,
	})

	resources, err := buildResources(t)
	if err != nil {
		return err
	}

	hc := container.HostConfig{
		PublishAllPorts: true,
		Mounts:          mounts,
		Resources:       resources,
	}

	cc := buildContainerConfig(t, workdir.Target, env)
	nc := buildNetworkConfig(t)

	resp, err := d.client.ContainerCreate(
		ctx, &cc, &hc, &nc, nil, "")
	if err != nil {
		flog.Error(fmt.Errorf("error creating container using image %s: %w", t.Image, err))
		return err
	}

	d.tasks.Set(t.ID, resp.ID)
	flog.Info("created container %s", resp.ID)

	defer func() {
		stopContext, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		if err := d.Stop(stopContext, t); err != nil {
			flog.Error(fmt.Errorf("container-id: %s, error removing container upon completion, %w", resp.ID, err))
		}
	}()

	if err := d.initWorkdir(ctx, resp.ID, t); err != nil {
		return fmt.Errorf("error initializing container, %w", err)
	}

	flog.Info("Starting container %s", resp.ID)
	err = d.client.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		return fmt.Errorf("error starting container %s: %w", resp.ID, err)
	}

	out, err := d.client.ContainerLogs(ctx, resp.ID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	})
	if err != nil {
		return fmt.Errorf("error getting logs for container %s: %w", resp.ID, err)
	}
	defer func() {
		if err := out.Close(); err != nil {
			flog.Error(fmt.Errorf("error closing stdout on container %s, %w", resp.ID, err))
		}
	}()
	_, err = io.Copy(os.Stdout, out)
	if err != nil {
		return fmt.Errorf("error reading the std out, %w", err)
	}

	return d.waitForCompletion(ctx, t, resp.ID)
}

func buildEnvVars(t *types.Task) []string {
	env := []string{"OUTPUT=/flowbot/stdout"}
	for name, value := range t.Env {
		env = append(env, fmt.Sprintf("%s=%s", name, value))
	}
	return env
}

func buildContainerMounts(t *types.Task) ([]mount.Mount, error) {
	var mounts []mount.Mount
	for _, m := range t.Mounts {
		var mt mount.Type
		switch m.Type {
		case types.MountTypeVolume:
			mt = mount.TypeVolume
			if m.Target == "" {
				return nil, fmt.Errorf("volume target is required")
			}
		case types.MountTypeBind:
			mt = mount.TypeBind
			if m.Target == "" {
				return nil, fmt.Errorf("bind target is required")
			}
			if m.Source == "" {
				return nil, fmt.Errorf("bind source is required")
			}
		case types.MountTypeTmpfs:
			mt = mount.TypeTmpfs
		default:
			return nil, fmt.Errorf("unknown mount type: %s", m.Type)
		}
		item := mount.Mount{
			Type:   mt,
			Source: m.Source,
			Target: m.Target,
		}
		flog.Info("Mounting %s -> %s", item.Source, item.Target)
		mounts = append(mounts, item)
	}
	return mounts, nil
}

func buildResources(t *types.Task) (container.Resources, error) {
	cpus, err := parseCPUs(t.Limits)
	if err != nil {
		return container.Resources{}, fmt.Errorf("invalid CPUs value, %w", err)
	}
	mem, err := parseMemory(t.Limits)
	if err != nil {
		return container.Resources{}, fmt.Errorf("invalid memory value, %w", err)
	}
	resources := container.Resources{
		NanoCPUs: cpus,
		Memory:   mem,
	}
	if t.GPUs != "" {
		gpuOpts := cliopts.GpuOpts{}
		if err := gpuOpts.Set(t.GPUs); err != nil {
			return container.Resources{}, fmt.Errorf("error setting GPUs, %w", err)
		}
		// resources.DeviceRequests = gpuOpts.Value() FIXME
	}
	return resources, nil
}

func buildContainerConfig(t *types.Task, workdirTarget string, env []string) container.Config {
	cmd := t.CMD
	if len(cmd) == 0 {
		cmd = []string{fmt.Sprintf("%s/entrypoint", workdirTarget)}
	}
	entrypoint := t.Entrypoint
	if len(entrypoint) == 0 && t.Run != "" {
		entrypoint = []string{"sh", "-c"}
	}
	cc := container.Config{
		Image:      t.Image,
		Env:        env,
		Cmd:        cmd,
		Entrypoint: entrypoint,
	}
	if len(t.Files) > 0 {
		cc.WorkingDir = workdirTarget
	}
	return cc
}

func buildNetworkConfig(t *types.Task) network.NetworkingConfig {
	nc := network.NetworkingConfig{
		EndpointsConfig: make(map[string]*network.EndpointSettings),
	}
	for _, nw := range t.Networks {
		nc.EndpointsConfig[nw] = &network.EndpointSettings{NetworkID: nw}
	}
	return nc
}

func (d *Runtime) waitForCompletion(ctx context.Context, t *types.Task, containerID string) error {
	if _, ok := d.tasks.Get(t.ID); !ok {
		return fmt.Errorf("task %s was stopped", t.ID)
	}
	statusCh, errCh := d.client.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	var status container.WaitResponse
	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
		status = <-statusCh
	case status = <-statusCh:
	}
	if status.StatusCode != 0 {
		out, err := d.client.ContainerLogs(ctx, containerID, container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Tail:       "10",
		})
		if err != nil {
			flog.Error(fmt.Errorf("error tailing the log, %w", err))
			return fmt.Errorf("exit code %d, %w", status.StatusCode, err)
		}
		buf, err := io.ReadAll(printableReader{reader: out})
		if err != nil {
			flog.Error(fmt.Errorf("error copying the output, %w", err))
		}
		return fmt.Errorf("exit code %d: %s", status.StatusCode, string(buf))
	}
	stdout, err := d.readOutput(ctx, containerID)
	if err != nil {
		return err
	}
	t.Result = stdout
	flog.Info("task-i: %s status-code: %d, task completed", t.ID, status.StatusCode)
	return nil
}

func (d *Runtime) readOutput(ctx context.Context, containerID string) (string, error) {
	r, _, err := d.client.CopyFromContainer(ctx, containerID, "/flowbot/stdout")
	if err != nil {
		return "", err
	}
	defer func() {
		err := r.Close()
		if err != nil {
			flog.Error(fmt.Errorf("error closing /flowbot/stdout reader, %w", err))
		}
	}()
	tr := tar.NewReader(r)
	var buf bytes.Buffer
	for {
		_, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return "", err
		}

		const maxOutputSize = 10 * 1024 * 1024 // 10 MB total
		remaining := maxOutputSize - int64(buf.Len())
		if remaining <= 0 {
			continue
		}
		if _, err := io.CopyN(&buf, tr, remaining); err != nil && !errors.Is(err, io.EOF) {
			return "", err
		}
	}
	return buf.String(), nil
}

func (d *Runtime) initWorkdir(ctx context.Context, containerID string, t *types.Task) error {
	flog.Info("initialize the workdir for container %s", containerID)
	// create the archive
	filename, err := createArchive(t)
	if err != nil {
		return err
	}
	// clean up the archive
	defer func() {
		err := os.Remove(filename)
		if err != nil {
			flog.Error(fmt.Errorf("error removing archive: %s, %w", filename, err))
		}
	}()
	// open the archive for reading by the container
	ar, err := os.Open(filename)
	if err != nil {
		return err
	}
	// close the archive
	defer func() {
		err := ar.Close()
		if err != nil {
			flog.Error(fmt.Errorf("error closing archive file, %w", err))
		}
	}()
	r := bufio.NewReader(ar)
	if err := d.client.CopyToContainer(ctx, containerID, "/flowbot", r, container.CopyToContainerOptions{}); err != nil {
		return err
	}
	return nil
}

func createArchive(t *types.Task) (string, error) {
	// create an archive file
	ar, err := os.CreateTemp("", "archive-*.tar")
	if err != nil {
		return "", fmt.Errorf("error creating tar file: %w", err)
	}
	defer func() {
		if err := ar.Close(); err != nil {
			flog.Error(fmt.Errorf("error closing archive.tar file, %w", err))
		}
	}()
	// write the run script as an entrypoint
	buf := bufio.NewWriter(ar)
	tw := tar.NewWriter(buf)
	if t.Run != "" {
		hdr := &tar.Header{
			Name: "entrypoint",
			Mode: 0111, // execute only
			Size: int64(len(t.Run)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return "", err
		}
		if _, err := tw.Write([]byte(t.Run)); err != nil {
			return "", err
		}
	}
	// write an stdout placeholder file
	hdr := &tar.Header{
		Name: "stdout",
		Mode: 0222, // write-only
		Size: int64(0),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return "", err
	}
	if _, err := tw.Write([]byte{}); err != nil {
		return "", err
	}
	// write all other files specified on the task
	for filename, contents := range t.Files {
		hdr := &tar.Header{
			Name: filename,
			Mode: 0444, // read-only
			Size: int64(len(contents)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return "", err
		}
		if _, err := tw.Write([]byte(contents)); err != nil {
			return "", err
		}
	}
	if err := buf.Flush(); err != nil {
		return "", err
	}
	// close the tar writer
	if err := tw.Close(); err != nil {
		return "", err
	}
	return ar.Name(), nil
}

func (d *Runtime) Stop(ctx context.Context, t *types.Task) error {
	if cancel, ok := d.cancels.Get(t.ID); ok {
		cancel()
	}
	containerID, ok := d.tasks.LoadAndDelete(t.ID)
	if !ok {
		return nil
	}
	flog.Info("Attempting to stop and remove container %v", containerID)
	return d.client.ContainerRemove(ctx, containerID, container.RemoveOptions{
		RemoveVolumes: true,
		RemoveLinks:   false,
		Force:         true,
	})
}

func (d *Runtime) HealthCheck(ctx context.Context) error {
	_, err := d.client.ContainerList(ctx, container.ListOptions{})
	return err
}

// take from https://github.com/docker/cli/blob/9bd5ec504afd13e82d5e50b60715e7190c1b2aa0/opts/opts.go#L393-L403
func parseCPUs(limits *types.TaskLimits) (int64, error) {
	if limits == nil || limits.CPUs == "" {
		return 0, nil
	}
	cpu, ok := big.NewRat(0, 1).SetString(limits.CPUs)
	if !ok {
		return 0, fmt.Errorf("failed to parse %v as a rational number", limits.CPUs)
	}
	nano := cpu.Mul(cpu, big.NewRat(1e9, 1))
	if !nano.IsInt() {
		return 0, fmt.Errorf("value is too precise")
	}
	return nano.Num().Int64(), nil
}

func parseMemory(limits *types.TaskLimits) (int64, error) {
	if limits == nil || limits.Memory == "" {
		return 0, nil
	}
	return units.RAMInBytes(limits.Memory)
}

func (r printableReader) Read(p []byte) (int, error) {
	buf := make([]byte, len(p))
	n, err := r.reader.Read(buf)
	if err != nil {
		if err != io.EOF {
			return 0, err
		}
	}
	j := 0
	for i := 0; i < n; {
		ru, size := utf8.DecodeRune(buf[i:n])
		if ru == utf8.RuneError && size <= 1 {
			i += size
			continue
		}
		if unicode.IsPrint(ru) || ru == '\n' || ru == '\r' || ru == '\t' {
			j += copy(p[j:], buf[i:i+size])
		}
		i += size
	}
	return j, err
}

func (d *Runtime) imagePull(ctx context.Context, t *types.Task) error {
	_, ok := d.images.Get(t.Image)
	if ok {
		return nil
	}
	// Acquire per-image lock to deduplicate concurrent pulls of the same image.
	mu, _ := d.pullMu.LoadOrStore(t.Image, &sync.Mutex{})
	mu.Lock()
	defer mu.Unlock()
	// Double-check after acquiring the lock; another goroutine may have pulled it.
	if _, ok := d.images.Get(t.Image); ok {
		return nil
	}
	// Check if the image exists locally.
	images, err := d.client.ImageList(
		ctx,
		image.ListOptions{All: true},
	)
	if err != nil {
		return err
	}
	for _, img := range images {
		for _, tag := range img.RepoTags {
			if tag == t.Image {
				d.images.Set(tag, true)
				return nil
			}
		}
	}
	pr := &pullRequest{
		image: t.Image,
		done:  make(chan error),
	}
	if t.Registry != nil {
		pr.registry = registry{
			username: t.Registry.Username,
			password: t.Registry.Password,
		}
	}
	d.pullq <- pr
	err = <-pr.done
	if err == nil {
		d.images.Set(t.Image, true)
	}
	return err
}

// puller is a goroutine that serializes all requests
// to pull images from the docker repo
func (d *Runtime) puller() {
	defer close(d.pullerDone)
	for pr := range d.pullq {
		var authConfig regtypes.AuthConfig
		if pr.registry.username != "" {
			authConfig = regtypes.AuthConfig{
				Username: pr.registry.username,
				Password: pr.registry.password,
			}
		} else {
			ref, err := parseRef(pr.image)
			if err != nil {
				pr.done <- err
				continue
			}
			if ref.domain != "" {
				username, password, err := getRegistryCredentials(d.config, ref.domain)
				if err != nil {
					pr.done <- err
					continue
				}
				authConfig = regtypes.AuthConfig{
					Username: username,
					Password: password,
				}
			}
		}

		encodedJSON, err := sonic.Marshal(authConfig)
		if err != nil {
			pr.done <- err
			continue
		}
		authStr := base64.URLEncoding.EncodeToString(encodedJSON)
		reader, err := d.client.ImagePull(
			context.Background(), pr.image, image.PullOptions{RegistryAuth: authStr})
		if err != nil {
			pr.done <- err
			continue
		}
		_, err = io.Copy(os.Stdout, reader)
		if err != nil {
			pr.done <- err
			continue
		}
		pr.done <- nil
	}
}

// Close shuts down the Docker runtime, stopping the puller goroutine
// and closing the Docker client connection.
func (d *Runtime) Close() error {
	close(d.pullq)
	<-d.pullerDone
	return d.client.Close()
}
