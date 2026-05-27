package shell

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/flowline-io/flowbot/pkg/utils/reexec"
	"github.com/flowline-io/flowbot/pkg/utils/syncx"
)

type Rexec func(args ...string) *exec.Cmd

const (
	DefaultUid   = "-"
	DefaultGid   = "-"
	envVarPrefix = "REEXEC_"
)

// Register registers the shell reexec handler for self-reexecution.
func Register() {
	reexec.Register("shell", reexecRun)
}

type Runtime struct {
	cmds   *syncx.Map[string, *exec.Cmd]
	shell  []string
	uid    string
	gid    string
	reexec Rexec
}

type Config struct {
	CMD   []string
	UID   string
	GID   string
	Rexec Rexec
}

func NewShellRuntime(cfg Config) *Runtime {
	if len(cfg.CMD) == 0 {
		cfg.CMD = []string{"bash", "-c"}
	}
	if cfg.Rexec == nil {
		cfg.Rexec = reexec.Command
	}
	if cfg.UID == "" {
		cfg.UID = DefaultUid
	}
	if cfg.GID == "" {
		cfg.GID = DefaultGid
	}
	return &Runtime{
		cmds:   new(syncx.Map[string, *exec.Cmd]),
		shell:  cfg.CMD,
		uid:    cfg.UID,
		gid:    cfg.GID,
		reexec: cfg.Rexec,
	}
}

func (r *Runtime) Run(ctx context.Context, t *types.Task) error {
	if err := validateTask(t); err != nil {
		return err
	}
	for _, pre := range t.Pre {
		pre.ID = utils.NewUUID()
		if err := r.doRun(ctx, pre); err != nil {
			return err
		}
	}
	if err := r.doRun(ctx, t); err != nil {
		return err
	}
	for _, post := range t.Post {
		post.ID = utils.NewUUID()
		if err := r.doRun(ctx, post); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runtime) doRun(ctx context.Context, t *types.Task) error {
	defer r.cmds.Delete(t.ID)

	workdir, err := r.setupWorkdir(t)
	if err != nil {
		return err
	}
	defer func() {
		_ = os.RemoveAll(workdir)
	}()
	flog.Debug("Created workdir %s", workdir)

	args, env, err := r.buildShellCommand(workdir, t)
	if err != nil {
		return err
	}

	cmd := r.reexec(args...)
	cmd.Env = env
	cmd.Dir = workdir
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	defer func() { _ = stdout.Close() }()
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return err
	}

	r.cmds.Set(t.ID, cmd)

	go readProcessOutput(stdout)

	errChan := make(chan error)
	doneChan := make(chan any)
	go func() {
		if err := cmd.Wait(); err != nil {
			errChan <- err
			return
		}
		close(doneChan)
	}()
	select {
	case err := <-errChan:
		return fmt.Errorf("error executing command, %w", err)
	case <-ctx.Done():
		if err := cmd.Process.Kill(); err != nil {
			return fmt.Errorf("error canceling command, %w", err)
		}
		return ctx.Err()
	case <-doneChan:
	}

	output, err := os.ReadFile(fmt.Sprintf("%s/stdout", workdir))
	if err != nil {
		return fmt.Errorf("error reading the task output, %w", err)
	}

	t.Result = string(output)

	return nil
}

// validateTask checks that the task does not contain fields unsupported by the shell runtime.
func validateTask(t *types.Task) error {
	if t.ID == "" {
		return errors.New("task id is required")
	}
	if len(t.Mounts) > 0 {
		return errors.New("mounts are not supported on shell runtime")
	}
	if len(t.Entrypoint) > 0 {
		return errors.New("entrypoint is not supported on shell runtime")
	}
	if t.Image != "" {
		return errors.New("image is not supported on shell runtime")
	}
	if t.Limits != nil && (t.Limits.CPUs != "" || t.Limits.Memory != "") {
		return errors.New("limits are not supported on shell runtime")
	}
	if len(t.Networks) > 0 {
		return errors.New("networks are not supported on shell runtime")
	}
	if t.Registry != nil {
		return errors.New("registry is not supported on shell runtime")
	}
	if len(t.CMD) > 0 {
		return errors.New("cmd is not supported on shell runtime")
	}
	return nil
}

// setupWorkdir creates a temporary work directory and writes task files into it.
func (*Runtime) setupWorkdir(t *types.Task) (string, error) {
	workdir, err := os.MkdirTemp("", fmt.Sprintf("flowbot-%s", t.ID))
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(fmt.Sprintf("%s/stdout", workdir), []byte{}, 0606); err != nil {
		return "", fmt.Errorf("error writing the entrypoint, %w", err)
	}

	for filename, contents := range t.Files {
		filename = fmt.Sprintf("%s/%s", workdir, filename)
		if err := os.WriteFile(filename, []byte(contents), 0444); err != nil {
			return "", fmt.Errorf("error writing file: %s, %w", filename, err)
		}
	}

	return workdir, nil
}

// buildShellCommand constructs the environment variables and reexec args for the shell command.
func (r *Runtime) buildShellCommand(workdir string, t *types.Task) ([]string, []string, error) {
	var env []string
	for name, value := range t.Env {
		env = append(env, fmt.Sprintf("%s%s=%s", envVarPrefix, name, value))
	}
	env = append(env,
		fmt.Sprintf("%sOUTPUT=%s/stdout", envVarPrefix, workdir),
		fmt.Sprintf("WORKDIR=%s", workdir),
		fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
	)

	if err := os.WriteFile(fmt.Sprintf("%s/entrypoint", workdir), []byte(t.Run), 0555); err != nil {
		return nil, nil, fmt.Errorf("error writing the entrypoint, %w", err)
	}

	args := make([]string, len(r.shell)+1)
	copy(args, r.shell)
	args[len(r.shell)] = fmt.Sprintf("%s/entrypoint", workdir)
	args = append([]string{"shell", "-uid", r.uid, "-gid", r.gid}, args...)
	return args, env, nil
}

// readProcessOutput reads and prints lines from the command's stdout.
func readProcessOutput(stdout io.ReadCloser) {
	reader := bufio.NewReader(stdout)
	line, err := reader.ReadString('\n')
	for err == nil {
		_, _ = fmt.Println(line)
		line, err = reader.ReadString('\n')
	}
}

func reexecRun() {
	var uid string
	var gid string
	fs := flag.NewFlagSet("shell", flag.ExitOnError)
	fs.StringVar(&uid, "uid", "", "the uid to use when running the process")
	fs.StringVar(&gid, "gid", "", "the gid to use when running the process")
	_ = fs.Parse(os.Args[1:])

	SetUID(uid)
	SetGID(gid)

	workdir := os.Getenv("WORKDIR")
	if workdir == "" {
		flog.Error(errors.New("work dir not set"))
		return
	}

	var env []string
	for _, entry := range os.Environ() {
		kv := strings.Split(entry, "=")
		if len(kv) != 2 {
			flog.Error(fmt.Errorf("invalid env var: %s", entry))
		}
		if after, ok := strings.CutPrefix(kv[0], envVarPrefix); ok {
			k := after
			v := kv[1]
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	cmd := exec.Command(fs.Args()[0], fs.Args()[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env
	cmd.Dir = workdir

	flog.Info("reexecing: %s as %s:%s", strings.Join(fs.Args(), " "), uid, gid)
	if err := cmd.Run(); err != nil {
		flog.Error(fmt.Errorf("error reexecing: %s", strings.Join(fs.Args(), " ")))
	}
}

func (r *Runtime) Stop(_ context.Context, t *types.Task) error {
	proc, ok := r.cmds.Get(t.ID)
	if !ok {
		return nil
	}
	if err := proc.Process.Kill(); err != nil {
		return fmt.Errorf("error stopping process for task: %s, %w", t.ID, err)
	}
	return nil
}

func (*Runtime) HealthCheck(_ context.Context) error {
	return nil
}

func (*Runtime) Close() error {
	return nil
}
