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
	"time"
	"unicode"

	cliopts "github.com/docker/cli/opts"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	regtypes "github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/docker/go-units"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/executer/runtime"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/flowline-io/flowbot/pkg/utils/syncx"
	jsoniter "github.com/json-iterator/go"
)

type Runtime struct {
	client  *client.Client
	tasks   *syncx.Map[string, string]
	images  *syncx.Map[string, bool]
	pullq   chan *pullRequest
	mounter runtime.Mounter
	config  string
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

func WithConfig(config string) Option {
	return func(rt *Runtime) {
		rt.config = config
	}
}

func NewRuntime(opts ...Option) (*Runtime, error) {
	dc, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}
	rt := &Runtime{
		client: dc,
		tasks:  new(syncx.Map[string, string]),
		images: new(syncx.Map[string, bool]),
		pullq:  make(chan *pullRequest, 1),
	}
	for _, o := range opts {
		o(rt)
	}
	// setup a default mounter
	if rt.mounter == nil {
		vmounter, err := NewVolumeMounter()
		if err != nil {
			return nil, err
		}
		rt.mounter = vmounter
	}
	go rt.puller(context.Background())
	return rt, nil
}

func (d *Runtime) Run(ctx context.Context, t *types.Task) error {
	// prepare mounts
	for i, mnt := range t.Mounts {
		err := d.mounter.Mount(ctx, &mnt)
		if err != nil {
			return err
		}
		defer func(m types.Mount) {//revive:disable
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
	if err := d.imagePull(ctx, t); err != nil {
		return fmt.Errorf("error pulling image: %s, %w", t.Image, err)
	}

	var env []string
	for name, value := range t.Env {
		env = append(env, fmt.Sprintf("%s=%s", name, value))
	}
	env = append(env, "OUTPUT=/flowbot/stdout")

	var mounts []mount.Mount

	for _, m := range t.Mounts {
		var mt mount.Type
		switch m.Type {
		case types.MountTypeVolume:
			mt = mount.TypeVolume
			if m.Target == "" {
				return fmt.Errorf("volume target is required")
			}
		case types.MountTypeBind:
			mt = mount.TypeBind
			if m.Target == "" {
				return fmt.Errorf("bind target is required")
			}
			if m.Source == "" {
				return fmt.Errorf("bind source is required")
			}
		case types.MountTypeTmpfs:
			mt = mount.TypeTmpfs
		default:
			return fmt.Errorf("unknown mount type: %s", m.Type)
		}
		item := mount.Mount{
			Type:   mt,
			Source: m.Source,
			Target: m.Target,
		}
		flog.Debug("Mounting %s -> %s", item.Source, item.Target)
		mounts = append(mounts, item)
	}
	// create the workdir mount
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

	// parse task limits
	cpus, err := parseCPUs(t.Limits)
	if err != nil {
		return fmt.Errorf("invalid CPUs value, %w", err)
	}
	mem, err := parseMemory(t.Limits)
	if err != nil {
		return fmt.Errorf("invalid memory value, %w", err)
	}

	resources := container.Resources{
		NanoCPUs: cpus,
		Memory:   mem,
	}

	if t.GPUs != "" {
		gpuOpts := cliopts.GpuOpts{}
		if err := gpuOpts.Set(t.GPUs); err != nil {
			return fmt.Errorf("error setting GPUs, %w", err)
		}
		resources.DeviceRequests = gpuOpts.Value()
	}

	hc := container.HostConfig{
		PublishAllPorts: true,
		Mounts:          mounts,
		Resources:       resources,
	}

	cmd := t.CMD
	if len(cmd) == 0 {
		cmd = []string{fmt.Sprintf("%s/entrypoint", workdir.Target)}
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
	// we want to override the default
	// image WORKDIR only if the task
	// introduces work files
	if len(t.Files) > 0 {
		cc.WorkingDir = workdir.Target
	}

	nc := network.NetworkingConfig{
		EndpointsConfig: make(map[string]*network.EndpointSettings),
	}

	for _, nw := range t.Networks {
		nc.EndpointsConfig[nw] = &network.EndpointSettings{NetworkID: nw}
	}

	resp, err := d.client.ContainerCreate(
		ctx, &cc, &hc, &nc, nil, "")
	if err != nil {
		flog.Error(fmt.Errorf("Error creating container using image %s: %w", t.Image, err))
		return err
	}

	// create a mapping between task id and container id
	d.tasks.Set(t.ID, resp.ID)

	flog.Debug("created container %s", resp.ID)

	// remove the container
	defer func() {
		stopContext, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		if err := d.Stop(stopContext, t); err != nil {
			flog.Error(fmt.Errorf("container-id: %s, error removing container upon completion, %w", resp.ID, err))
		}
	}()

	// initialize the work directory
	if err := d.initWorkdir(ctx, resp.ID, t); err != nil {
		return fmt.Errorf("error initializing container, %w", err)
	}

	// start the container
	flog.Debug("Starting container %s", resp.ID)
	err = d.client.ContainerStart(
		ctx, resp.ID, container.StartOptions{})
	if err != nil {
		return fmt.Errorf("error starting container %s: %w", resp.ID, err)
	}
	// read the container's stdout
	out, err := d.client.ContainerLogs(
		ctx,
		resp.ID,
		container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     true,
		},
	)
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
	statusCh, errCh := d.client.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
	case status := <-statusCh:
		if status.StatusCode != 0 { // error
			out, err := d.client.ContainerLogs(
				ctx,
				resp.ID,
				container.LogsOptions{
					ShowStdout: true,
					ShowStderr: true,
					Tail:       "10",
				},
			)
			if err != nil {
				flog.Error(fmt.Errorf("error tailing the log, %w", err))
				return fmt.Errorf("exit code %d, %w", status.StatusCode, err)
			}
			buf, err := io.ReadAll(printableReader{reader: out})
			if err != nil {
				flog.Error(fmt.Errorf("error copying the output, %w", err))
			}
			return fmt.Errorf("exit code %d: %s", status.StatusCode, string(buf))
		} else {
			stdout, err := d.readOutput(ctx, resp.ID)
			if err != nil {
				return err
			}
			t.Result = stdout
		}
		flog.Debug("task-i: %s status-code: %d, task completed", t.ID, status.StatusCode)
	}
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

		if _, err := io.Copy(&buf, tr); err != nil {
			return "", err
		}
	}
	return buf.String(), nil
}

func (d *Runtime) initWorkdir(ctx context.Context, containerID string, t *types.Task) error {
	flog.Debug("initialize the workdir for container %s", containerID)
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
		return "", fmt.Errorf("error creating tar file")
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
	containerID, ok := d.tasks.Get(t.ID)
	if !ok {
		return nil
	}
	d.tasks.Delete(t.ID)
	flog.Debug("Attempting to stop and remove container %v", containerID)
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
	cpu, ok := new(big.Rat).SetString(limits.CPUs)
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
	for i := 0; i < n; i++ {
		if unicode.IsPrint(rune(buf[i])) {
			p[j] = buf[i]
			j++
		}
	}
	return j, err
}

func (d *Runtime) imagePull(ctx context.Context, t *types.Task) error {
	_, ok := d.images.Get(t.Image)
	if ok {
		return nil
	}
	// let's check if we have the image
	// locally already
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
	return <-pr.done
}

// puller is a goroutine that serializes all requests
// to pull images from the docker repo
func (d *Runtime) puller(ctx context.Context) {
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

		encodedJSON, err := jsoniter.Marshal(authConfig)
		if err != nil {
			pr.done <- err
			continue
		}
		authStr := base64.URLEncoding.EncodeToString(encodedJSON)
		reader, err := d.client.ImagePull(
			ctx, pr.image, image.PullOptions{RegistryAuth: authStr})
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
