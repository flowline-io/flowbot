package sandbox

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/flowline-io/flowbot/pkg/agent/env"
	"github.com/flowline-io/flowbot/pkg/executor/runtime/kern"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils"
)

const sandboxRuntimeKern = "kern"

// KernRunner runs commands via the kern CLI.
type KernRunner struct {
	Client          *kern.Client
	SecurityProfile string
}

// Run starts an ephemeral kern box, waits for exit, and returns captured output.
func (r KernRunner) Run(ctx context.Context, opts RunOptions) (env.Capture, error) {
	if err := validateRunOptions(opts); err != nil {
		return env.Capture{}, err
	}
	if err := validateKernNetwork(opts.Network); err != nil {
		return env.Capture{}, err
	}

	opts, cleanupDirs, err := prepareKernRunOptions(opts)
	if err != nil {
		return env.Capture{}, err
	}
	defer cleanupTempDirs(cleanupDirs)

	cmd, err := buildCommand(opts)
	if err != nil {
		return env.Capture{}, err
	}

	client := r.client()
	boxName := kern.BoxName(utils.NewUUID())
	workDir := kernWorkDir(opts)
	boxOpts := buildKernBoxOptions(opts, boxName, workDir, cmd, r.SecurityProfile)

	if err := client.Pull(ctx, opts.Image); err != nil {
		return env.Capture{}, err
	}
	defer stopKernBox(client, boxName)

	flog.Info("[sandbox] kern box create image=%s workspace=%s workdir=%s cli_creds=%s %s",
		opts.Image, opts.Workspace, workDir, cliCredsLabel(opts.AccessToken), summarizeCommand(opts))

	stdout, stderr, exitCode, err := client.BoxJob(ctx, boxOpts)
	if err != nil {
		return env.Capture{}, err
	}
	if ctx.Err() != nil {
		return env.Capture{}, ctx.Err()
	}
	flog.Info("[sandbox] kern box done name=%s workspace=%s workdir=%s exit_code=%d",
		boxName, opts.Workspace, workDir, exitCode)
	return env.Capture{Stdout: stdout, Stderr: stderr, ExitCode: exitCode}, nil
}

func prepareKernRunOptions(opts RunOptions) (RunOptions, []string, error) {
	var cleanupDirs []string
	if opts.AccessToken != "" && opts.CLIConfigDir == "" {
		dir, err := materializeCLIConfig(opts.ServerURL, opts.AccessToken)
		if err != nil {
			return opts, nil, err
		}
		opts.CLIConfigDir = dir
		cleanupDirs = append(cleanupDirs, dir)
	}
	if dir := injectCLIBinary(&opts); dir != "" {
		cleanupDirs = append(cleanupDirs, dir)
	}
	return opts, cleanupDirs, nil
}

func cleanupTempDirs(dirs []string) {
	for _, d := range dirs {
		_ = os.RemoveAll(d)
	}
}

func kernWorkDir(opts RunOptions) string {
	if opts.WorkDir != "" {
		return opts.WorkDir
	}
	return containerWorkspacePath(opts.Workspace)
}

func buildKernBoxOptions(opts RunOptions, boxName, workDir string, cmd []string, securityProfile string) kern.BoxOptions {
	binds := []string{kern.BindSpec(opts.Workspace, containerWorkspacePath(opts.Workspace), false)}
	if opts.CLIConfigDir != "" {
		binds = append(binds, kern.BindSpec(opts.CLIConfigDir, containerCLIConfigPath, true))
	}
	if opts.CLIBinaryDir != "" {
		binds = append(binds, kern.BindSpec(opts.CLIBinaryDir, containerCLIDirPath, true))
	}
	return kern.BoxOptions{
		Name:            boxName,
		Image:           opts.Image,
		Command:         cmd,
		Env:             buildContainerEnv(opts),
		Binds:           binds,
		Memory:          opts.Memory,
		Network:         opts.Network,
		Workdir:         workDir,
		SecurityProfile: securityProfile,
	}
}

func stopKernBox(client *kern.Client, boxName string) {
	stopCtx, cancel := context.WithTimeout(context.Background(), defaultStopWait)
	defer cancel()
	if err := client.Stop(stopCtx, boxName); err != nil {
		flog.Warn("[sandbox] kern stop failed name=%s err=%s", boxName, err.Error())
	}
}

func (r KernRunner) client() *kern.Client {
	if r.Client != nil {
		return r.Client
	}
	return kern.NewClient(kern.Config{SecurityProfile: r.SecurityProfile})
}

func selectRunner(cfg Config) Runner {
	if strings.EqualFold(strings.TrimSpace(cfg.Runtime), sandboxRuntimeKern) {
		return KernRunner{SecurityProfile: strings.TrimSpace(cfg.SecurityProfile)}
	}
	return DockerRunner{}
}

func sandboxRuntimeLabel(runtime string) string {
	if strings.EqualFold(strings.TrimSpace(runtime), sandboxRuntimeKern) {
		return sandboxRuntimeKern
	}
	return "docker"
}

func validateKernNetwork(network string) error {
	network = strings.TrimSpace(network)
	switch network {
	case "", "host", "none":
		return nil
	default:
		return fmt.Errorf("sandbox: kern runtime supports network %q only as empty, host, or none", network)
	}
}

func warnKernDockerInternal(serverURL string) {
	if !needsHostGateway(serverURL) {
		return
	}
	flog.Warn("[sandbox] kern runtime cannot resolve host.docker.internal; use server_url http://127.0.0.1:6060 with network: host")
}
