package kern

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/flowline-io/flowbot/pkg/utils/syncx"
)

const workdirMountTarget = "/flowbot"

// Runtime executes workflow tasks via the kern CLI.
type Runtime struct {
	client      *Client
	cfg         Config
	boxes       *syncx.Map[string, string]
	bindAllowed bool
}

// Option configures Runtime construction.
type Option func(*Runtime)

// WithConfig sets runtime configuration.
func WithConfig(cfg Config) Option {
	return func(rt *Runtime) {
		rt.cfg = cfg
		rt.bindAllowed = cfg.BindAllowed
	}
}

// WithClient sets a pre-built Client (tests).
func WithClient(client *Client) Option {
	return func(rt *Runtime) {
		rt.client = client
	}
}

// NewRuntime creates a kern workflow runtime.
func NewRuntime(opts ...Option) (*Runtime, error) {
	rt := &Runtime{
		boxes: new(syncx.Map[string, string]),
	}
	for _, o := range opts {
		o(rt)
	}
	if rt.client == nil {
		rt.client = NewClient(rt.cfg)
	}
	if err := LookPath(rt.client.Binary()); err != nil {
		return nil, fmt.Errorf("kern binary not found: %w", err)
	}
	return rt, nil
}

// Run executes the task in a one-shot kern box.
func (r *Runtime) Run(ctx context.Context, t *types.Task) error {
	for _, pre := range t.Pre {
		pre.ID = utils.NewUUID()
		pre.Mounts = t.Mounts
		pre.Networks = t.Networks
		pre.Limits = t.Limits
		if err := r.doRun(ctx, pre); err != nil {
			return err
		}
	}
	if err := r.doRun(ctx, t); err != nil {
		return err
	}
	for _, post := range t.Post {
		post.ID = utils.NewUUID()
		post.Mounts = t.Mounts
		post.Networks = t.Networks
		post.Limits = t.Limits
		if err := r.doRun(ctx, post); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runtime) doRun(ctx context.Context, t *types.Task) error {
	if t.ID == "" {
		return errors.New("task id is required")
	}
	if err := validateTask(t, r.bindAllowed); err != nil {
		return err
	}

	boxName := BoxName(t.ID)
	r.boxes.Set(t.ID, boxName)
	defer func() {
		r.boxes.Delete(t.ID)
		stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := r.client.Stop(stopCtx, boxName); err != nil {
			flog.Error(fmt.Errorf("kern stop %s: %w", boxName, err))
		}
	}()

	if err := r.client.Pull(ctx, t.Image); err != nil {
		return fmt.Errorf("error pulling image %s: %w", t.Image, err)
	}

	workdir, err := setupWorkdir(t)
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(workdir) }()

	binds, err := buildBinds(workdir, t)
	if err != nil {
		return err
	}

	cmd, entrypoint, workdirInBox := buildCommand(t)
	env := buildEnv(t)

	opts := BoxOptions{
		Name:            boxName,
		Image:           t.Image,
		Command:         cmd,
		Entrypoint:      entrypoint,
		Env:             env,
		Binds:           binds,
		Memory:          limitsMemory(t),
		CPUs:            limitsCPUs(t),
		Workdir:         workdirInBox,
		SecurityProfile: r.cfg.SecurityProfile,
		RequireLimits:   r.cfg.RequireLimits,
	}

	_, stderr, exitCode, err := r.client.BoxJob(ctx, opts)
	if err != nil {
		return err
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if exitCode != 0 {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = fmt.Sprintf("exit code %d", exitCode)
		}
		return fmt.Errorf("exit code %d: %s", exitCode, msg)
	}

	output, err := os.ReadFile(filepath.Join(workdir, "stdout"))
	if err != nil {
		return fmt.Errorf("read task output: %w", err)
	}
	t.Result = string(output)
	return nil
}

// Stop stops a running kern box for the task.
func (r *Runtime) Stop(_ context.Context, t *types.Task) error {
	name, ok := r.boxes.LoadAndDelete(t.ID)
	if !ok {
		return nil
	}
	stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return r.client.Stop(stopCtx, name)
}

// HealthCheck verifies kern is usable on the host.
func (r *Runtime) HealthCheck(ctx context.Context) error {
	return r.client.Doctor(ctx)
}

// Close releases runtime resources.
func (*Runtime) Close() error {
	return nil
}

func validateTask(t *types.Task, bindAllowed bool) error {
	if t.Image == "" {
		return errors.New("image is required for kern runtime")
	}
	if t.Registry != nil && (t.Registry.Username != "" || t.Registry.Password != "") {
		return errors.New("kern runtime v1 does not support registry credentials; use public images or kern login on the host")
	}
	if t.GPUs != "" {
		return errors.New("gpus are not supported on kern runtime")
	}
	if len(t.Networks) > 0 {
		return errors.New("networks are not supported on kern runtime")
	}
	for _, mnt := range t.Mounts {
		switch mnt.Type {
		case types.MountTypeBind:
			if err := ValidateBindMount(bindAllowed, mnt); err != nil {
				return err
			}
		case types.MountTypeVolume, types.MountTypeTmpfs:
			return fmt.Errorf("mount type %q is not supported on kern runtime", mnt.Type)
		default:
			return fmt.Errorf("unknown mount type: %s", mnt.Type)
		}
	}
	return nil
}

func buildBinds(workdir string, t *types.Task) ([]string, error) {
	binds := []string{BindSpec(workdir, workdirMountTarget, false)}
	for _, mnt := range t.Mounts {
		if err := PrepareBindMount(context.Background(), &mnt); err != nil {
			return nil, err
		}
		binds = append(binds, BindSpec(mnt.Source, mnt.Target, false))
	}
	return binds, nil
}

func setupWorkdir(t *types.Task) (string, error) {
	workdir, err := os.MkdirTemp("", fmt.Sprintf("flowbot-kern-%s-", BoxName(t.ID)))
	if err != nil {
		return "", err
	}
	if t.Run != "" {
		if err := os.WriteFile(filepath.Join(workdir, "entrypoint"), []byte(t.Run), 0o111); err != nil {
			return "", fmt.Errorf("write entrypoint: %w", err)
		}
	}
	if err := os.WriteFile(filepath.Join(workdir, "stdout"), nil, 0o222); err != nil {
		return "", fmt.Errorf("write stdout placeholder: %w", err)
	}
	for name, contents := range t.Files {
		path := filepath.Join(workdir, name)
		clean := filepath.Clean(path)
		if clean != path || !strings.HasPrefix(clean+string(os.PathSeparator), workdir+string(os.PathSeparator)) {
			return "", fmt.Errorf("invalid file path: %s", name)
		}
		if err := os.MkdirAll(filepath.Dir(clean), 0o750); err != nil {
			return "", fmt.Errorf("mkdir for file %s: %w", name, err)
		}
		if err := os.WriteFile(clean, []byte(contents), 0o444); err != nil {
			return "", fmt.Errorf("write file %s: %w", name, err)
		}
	}
	return workdir, nil
}

func buildCommand(t *types.Task) (cmd, entrypoint []string, workdir string) {
	cmd = t.CMD
	if len(cmd) == 0 {
		cmd = []string{workdirMountTarget + "/entrypoint"}
	}
	entrypoint = t.Entrypoint
	if len(entrypoint) == 0 && t.Run != "" {
		entrypoint = []string{"sh", "-c"}
	}
	if len(t.Files) > 0 || t.Run != "" {
		workdir = workdirMountTarget
	}
	return cmd, entrypoint, workdir
}

func buildEnv(t *types.Task) []string {
	env := []string{"OUTPUT=" + workdirMountTarget + "/stdout"}
	for name, value := range t.Env {
		env = append(env, fmt.Sprintf("%s=%s", name, value))
	}
	return env
}

func limitsMemory(t *types.Task) string {
	if t.Limits == nil {
		return ""
	}
	return t.Limits.Memory
}

func limitsCPUs(t *types.Task) string {
	if t.Limits == nil {
		return ""
	}
	return t.Limits.CPUs
}
