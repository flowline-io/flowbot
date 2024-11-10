package shell

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/flowline-io/flowbot/pkg/utils/reexec"
	"github.com/flowline-io/flowbot/pkg/utils/syncx"
	"github.com/rs/zerolog/log"
)

type Rexec func(args ...string) *exec.Cmd

const (
	DefaultUid   = "-"
	DefaultGid   = "-"
	envVarPrefix = "REEXEC_"
)

func init() {
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
	// excute pre-tasks
	for _, pre := range t.Pre {
		pre.ID = utils.NewUUID()
		if err := r.doRun(ctx, pre); err != nil {
			return err
		}
	}
	// run the actual task
	if err := r.doRun(ctx, t); err != nil {
		return err
	}
	// execute post tasks
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

	workdir, err := os.MkdirTemp("", "flowbot")
	if err != nil {
		return err
	}
	defer func() {
		_ = os.RemoveAll(workdir)
	}()

	log.Debug().Msgf("Created workdir %s", workdir)

	if err := os.WriteFile(fmt.Sprintf("%s/stdout", workdir), []byte{}, 0606); err != nil {
		return fmt.Errorf("error writing the entrypoint, %w", err)
	}

	for filename, contents := range t.Files {
		filename = fmt.Sprintf("%s/%s", workdir, filename)
		if err := os.WriteFile(filename, []byte(contents), 0444); err != nil {
			return fmt.Errorf("error writing file: %s, %w", filename, err)
		}
	}

	var env []string
	for name, value := range t.Env {
		env = append(env, fmt.Sprintf("%s%s=%s", envVarPrefix, name, value))
	}
	env = append(env, fmt.Sprintf("%sFLOWBOT_OUTPUT=%s/stdout", envVarPrefix, workdir))
	env = append(env, fmt.Sprintf("WORKDIR=%s", workdir))
	env = append(env, fmt.Sprintf("PATH=%s", os.Getenv("PATH")))

	if err := os.WriteFile(fmt.Sprintf("%s/entrypoint", workdir), []byte(t.Run), 0555); err != nil {
		return fmt.Errorf("error writing the entrypoint, %w", err)
	}
	args := append(r.shell, fmt.Sprintf("%s/entrypoint", workdir))
	args = append([]string{"shell", "-uid", r.uid, "-gid", r.gid}, args...)
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

	go func() {
		reader := bufio.NewReader(stdout)
		line, err := reader.ReadString('\n')
		for err == nil {
			_, _ = fmt.Println(line)
			line, err = reader.ReadString('\n')
		}
	}()

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

func reexecRun() {
	var uid string
	var gid string
	flag.StringVar(&uid, "uid", "", "the uid to use when running the process")
	flag.StringVar(&gid, "gid", "", "the gid to use when running the process")
	flag.Parse()

	SetUID(uid)
	SetGID(gid)

	workdir := os.Getenv("WORKDIR")
	if workdir == "" {
		log.Fatal().Msg("work dir not set")
	}

	var env []string
	for _, entry := range os.Environ() {
		kv := strings.Split(entry, "=")
		if len(kv) != 2 {
			log.Fatal().Msgf("invalid env var: %s", entry)
		}
		if strings.HasPrefix(kv[0], envVarPrefix) {
			k := strings.TrimPrefix(kv[0], envVarPrefix)
			v := kv[1]
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	cmd := exec.Command(flag.Args()[0], flag.Args()[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env
	cmd.Dir = workdir

	log.Debug().Msgf("reexecing: %s as %s:%s", strings.Join(flag.Args(), " "), uid, gid)
	if err := cmd.Run(); err != nil {
		log.Fatal().Err(err).Msgf("error reexecing: %s", strings.Join(flag.Args(), " "))
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

func (r *Runtime) HealthCheck(_ context.Context) error {
	return nil
}
