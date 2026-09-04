// Package sandbox provides optional Docker isolation for agent shell and code tools.
package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/containerd/errdefs"
	"github.com/docker/go-units"
	"github.com/flowline-io/flowbot/pkg/agent/env"
	"github.com/flowline-io/flowbot/pkg/agent/result"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

const (
	defaultImage            = "ghcr.io/flowline-io/flowbot-agent-sandbox:latest"
	defaultStopWait         = 5 * time.Second
	defaultCreateWait       = 2 * time.Minute
	maxLoggedCommandLen     = 200
	containerCLIConfigPath  = "/home/agent/.config/flowbot"
	containerCLIDirPath     = "/opt/flowbot-cli"
	containerCLIName        = "flowbot"
	siblingCLIBinaryName    = "flowbot-cli_linux_amd64"
	envFlowbotServerURL     = "FLOWBOT_SERVER_URL"
	envFlowbotToken         = "FLOWBOT_TOKEN"
	hostDockerInternal      = "host.docker.internal"
	cliConfigTempDirPattern = "flowbot-sandbox-cli-*"
	cliBinaryTempDirPattern = "flowbot-sandbox-cli-bin-*"
	cliTokenFileName        = "token"
	cliServerURLFileName    = "server_url"
	// cliExecOwner/cliExecWorld: execute/traverse; world 0755 is the chown-failed fallback.
	cliExecOwner          = 0750
	cliExecWorld          = 0755
	labelComponentKey     = "flowbot.component"
	labelComponentSandbox = "agent-sandbox"
	// containerInjectWorkDir is the container path used when WorkspaceInject copies the host workspace.
	containerInjectWorkDir = "/workspace"
	// sandboxAgentUID/GID match the agent user in deployments/agent-sandbox/Dockerfile.
	sandboxAgentUID = 1000
	sandboxAgentGID = 1000
	// cliConfigWorldReadable is used when chown to the sandbox agent fails (e.g. non-root host).
	// Legacy octal literals (0644) are used instead of 0o644 so swag can evaluate these consts.
	cliConfigWorldReadable = 0644
	cliConfigOwnerOnly     = 0600
	// Directory modes must include the execute bit so the agent can traverse into the config dir.
	cliConfigDirWorldAccessible = 0755
	cliConfigDirOwnerOnly       = 0700
)

var missingCLIBinaryWarn sync.Once

// executableDir is the server binary directory; tests override it for sibling-CLI resolution.
var executableDir = func() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exe), nil
}

// Config configures Docker sandbox execution.
type Config struct {
	// Runtime selects the sandbox backend (docker or kern). Empty defaults to docker.
	Runtime string
	// SecurityProfile sets kern --security-profile when Runtime is kern.
	SecurityProfile string
	// Image is the container image used for Exec.
	Image string
	// Network is the Docker network mode.
	Network string
	// Memory limits container memory (e.g. "512m").
	Memory string
	// Workspace is the host workspace path. By default it is bind-mounted at the same path
	// inside the container; when WorkspaceInject is set (Docker only), it is copied to /workspace.
	Workspace string
	// WorkspaceInject copies the host Workspace into /workspace via the Docker API instead of
	// bind-mounting. Ignored when Runtime is kern (bind mount remains).
	WorkspaceInject bool
	// ServerURL is the Flowbot API URL injected for the flowbot CLI inside the container.
	ServerURL string
	// AccessToken is the Hub access token injected for the flowbot CLI inside the container.
	AccessToken string
	// CLIPath is an optional configured host path to a linux/amd64 flowbot CLI binary.
	// Absolute paths are used as-is; relative paths are resolved beside the server executable.
	// When empty, New looks for flowbot-cli_linux_amd64 beside the server executable.
	CLIPath string
}

// ConfigFromChatAgent builds sandbox Config from chat agent settings.
func ConfigFromChatAgent(cfg config.ChatAgentSandboxConfig, workspace string) Config {
	image := strings.TrimSpace(cfg.Image)
	if image == "" {
		image = defaultImage
	}
	return Config{
		Runtime:         strings.TrimSpace(cfg.Runtime),
		SecurityProfile: strings.TrimSpace(cfg.SecurityProfile),
		Image:           image,
		Network:     strings.TrimSpace(cfg.Network),
		Memory:      strings.TrimSpace(cfg.Memory),
		Workspace:   strings.TrimSpace(workspace),
		ServerURL:   strings.TrimSpace(cfg.ServerURL),
		AccessToken: strings.TrimSpace(cfg.AccessToken),
		CLIPath:     strings.TrimSpace(cfg.CLIPath),
	}
}

// Runner executes a one-shot command inside a sandbox container.
type Runner interface {
	Run(ctx context.Context, opts RunOptions) (env.Capture, error)
}

// RunOptions configures one sandbox command invocation.
type RunOptions struct {
	Image       string
	Network     string
	Memory      string
	Workspace   string
	WorkDir     string
	Command     string
	Argv        []string
	ServerURL   string
	AccessToken string
	// Env is appended to the container environment (KEY=VALUE).
	Env []string
	// Stdin is process input; non-empty values are written to .flowbot-stdin under the host workdir.
	Stdin []byte
	// WorkspaceInject copies opts.Workspace into /workspace instead of bind-mounting (Docker only).
	WorkspaceInject bool
	// CLIConfigDir is a host directory bind-mounted read-only at containerCLIConfigPath.
	// When empty and AccessToken is set, DockerRunner materializes a temporary directory.
	CLIConfigDir string
	// CLIBinary is a host path to a linux/amd64 flowbot CLI file. DockerRunner copies it
	// into a temporary directory (as "flowbot") and bind-mounts that directory.
	CLIBinary string
	// CLIBinaryDir is a host directory bind-mounted read-only at containerCLIDirPath.
	// When empty and CLIBinary is set, DockerRunner materializes a temporary directory.
	CLIBinaryDir string
}

// Env implements env.ExecutionEnv with host filesystem ops and sandboxed Exec.
type Env struct {
	cfg       Config
	cliBinary string
	host      env.ExecutionEnv
	runner    Runner
}

// New creates a sandbox ExecutionEnv. Host FS ops use env.Default when host is nil.
func New(cfg Config, host env.ExecutionEnv, runner Runner) *Env {
	if host == nil {
		host = env.Default()
	}
	if runner == nil {
		runner = selectRunner(cfg)
	}
	if strings.EqualFold(strings.TrimSpace(cfg.Runtime), sandboxRuntimeKern) {
		warnKernDockerInternal(cfg.ServerURL)
	}
	cliBinary := ResolvedCLIBinary(cfg.CLIPath)
	creds := "none"
	if cfg.AccessToken != "" {
		creds = "injected"
	}
	cliBin := "none"
	if cliBinary != "" {
		cliBin = "injected"
	}
	flog.Info("[sandbox] env ready workspace=%s runtime=%s image=%s network=%s memory=%s cli_creds=%s cli_bin=%s",
		cfg.Workspace, sandboxRuntimeLabel(cfg.Runtime), cfg.Image, cfg.Network, cfg.Memory, creds, cliBin)
	return &Env{cfg: cfg, cliBinary: cliBinary, host: host, runner: runner}
}

// ResolvedCLIBinary returns an absolute host path to a regular file usable as the sandbox flowbot CLI.
// configured is chat_agent.sandbox.cli_path (may be empty). Missing or invalid paths warn once and return empty.
func ResolvedCLIBinary(configured string) string {
	path, err := candidateCLIPath(configured)
	if err != nil {
		warnMissingCLI("%s", err.Error())
		return ""
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		warnMissingCLI("resolve path %s: %s", path, err.Error())
		return ""
	}
	info, err := os.Stat(abs)
	if err != nil {
		warnMissingCLI("%s unavailable (%s); shell/code still work without flowbot", abs, err.Error())
		return ""
	}
	if info.IsDir() {
		warnMissingCLI("%s is a directory", abs)
		return ""
	}
	return abs
}

func candidateCLIPath(configured string) (string, error) {
	path := strings.TrimSpace(configured)
	if path != "" && filepath.IsAbs(path) {
		return path, nil
	}
	dir, err := executableDir()
	if err != nil {
		return "", fmt.Errorf("resolve server executable: %w", err)
	}
	if path == "" {
		return filepath.Join(dir, siblingCLIBinaryName), nil
	}
	return filepath.Join(dir, path), nil
}

func warnMissingCLI(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	missingCLIBinaryWarn.Do(func() {
		flog.Warn("[sandbox] flowbot CLI not injected: %s", msg)
	})
}

// ReadFile reads from the host filesystem.
func (e *Env) ReadFile(ctx context.Context, path string) result.Result[[]byte, result.FileError] {
	return e.host.ReadFile(ctx, path)
}

// WriteFile writes to the host filesystem.
func (e *Env) WriteFile(ctx context.Context, path string, data []byte, perm os.FileMode) result.Result[struct{}, result.FileError] {
	return e.host.WriteFile(ctx, path, data, perm)
}

// MkdirAll creates directories on the host filesystem.
func (e *Env) MkdirAll(ctx context.Context, path string, perm os.FileMode) result.Result[struct{}, result.FileError] {
	return e.host.MkdirAll(ctx, path, perm)
}

// Remove deletes a path on the host filesystem.
func (e *Env) Remove(ctx context.Context, path string) result.Result[struct{}, result.FileError] {
	return e.host.Remove(ctx, path)
}

// ReadDir lists directory entries on the host filesystem.
func (e *Env) ReadDir(ctx context.Context, path string) result.Result[[]env.DirEntry, result.FileError] {
	return e.host.ReadDir(ctx, path)
}

// Exec runs the command inside a sandbox container (bind-mounted or inject-copied workspace).
func (e *Env) Exec(ctx context.Context, opts env.ExecOptions) result.Result[env.Capture, result.ExecutionError] {
	runCtx := ctx
	if opts.Timeout != nil {
		runCtx = opts.Timeout
	}
	inject := workspaceInjectActive(e.cfg)
	workDir := resolveContainerWorkDir(e.cfg.Workspace, opts.Dir)
	if inject {
		workDir = containerInjectWorkDir
	}
	runOpts := RunOptions{
		Image:           e.cfg.Image,
		Network:         e.cfg.Network,
		Memory:          e.cfg.Memory,
		Workspace:       e.cfg.Workspace,
		WorkDir:         workDir,
		Command:         opts.Command,
		Argv:            append([]string(nil), opts.Argv...),
		ServerURL:       e.cfg.ServerURL,
		AccessToken:     e.cfg.AccessToken,
		Env:             append([]string(nil), opts.Env...),
		Stdin:           append([]byte(nil), opts.Stdin...),
		WorkspaceInject: inject,
		CLIBinary:       e.cliBinary,
	}
	if inject {
		runOpts.Env = remapHostPathsInEnv(runOpts.Env, e.cfg.Workspace, containerInjectWorkDir)
	}
	if ferr := e.applyStdinRedirect(ctx, opts, &runOpts); ferr != nil {
		return result.Err[env.Capture, result.ExecutionError](*ferr)
	}
	flog.Info("[sandbox] exec start workspace=%s workdir=%s cli_creds=%s %s",
		e.cfg.Workspace, workDir, cliCredsLabel(runOpts.AccessToken), summarizeCommand(runOpts))
	capture, err := e.runner.Run(runCtx, runOpts)
	if err != nil {
		code, cause := classifyExecError(runCtx, err)
		flog.Info("[sandbox] exec failed workspace=%s workdir=%s code=%s err=%s",
			e.cfg.Workspace, workDir, code, cause.Error())
		return result.Err[env.Capture, result.ExecutionError](
			result.NewExecutionError(code, cause.Error(), cause),
		)
	}
	flog.Info("[sandbox] exec done workspace=%s workdir=%s exit_code=%d",
		e.cfg.Workspace, workDir, capture.ExitCode)
	return result.Ok[env.Capture, result.ExecutionError](capture)
}

func workspaceInjectActive(cfg Config) bool {
	if !cfg.WorkspaceInject {
		return false
	}
	return !strings.EqualFold(strings.TrimSpace(cfg.Runtime), sandboxRuntimeKern)
}

func resolveContainerWorkDir(workspace, optsDir string) string {
	containerRoot := containerWorkspacePath(workspace)
	if optsDir == "" || workspace == "" {
		return containerRoot
	}
	rel, err := filepath.Rel(workspace, optsDir)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return containerRoot
	}
	return containerWorkspacePath(filepath.Join(workspace, rel))
}

func remapHostPathsInEnv(envVars []string, hostRoot, containerRoot string) []string {
	if len(envVars) == 0 || strings.TrimSpace(hostRoot) == "" {
		return envVars
	}
	hostRoot = containerWorkspacePath(hostRoot)
	out := make([]string, len(envVars))
	for i, entry := range envVars {
		key, val, ok := strings.Cut(entry, "=")
		if !ok {
			out[i] = entry
			continue
		}
		valNorm := containerWorkspacePath(val)
		switch {
		case valNorm == hostRoot:
			out[i] = key + "=" + containerRoot
		case strings.HasPrefix(valNorm, hostRoot+"/"):
			out[i] = key + "=" + containerRoot + strings.TrimPrefix(valNorm, hostRoot)
		default:
			out[i] = entry
		}
	}
	return out
}

func (e *Env) applyStdinRedirect(ctx context.Context, opts env.ExecOptions, runOpts *RunOptions) *result.ExecutionError {
	if len(opts.Stdin) == 0 {
		return nil
	}
	hostWorkDir := e.cfg.Workspace
	if opts.Dir != "" {
		hostWorkDir = opts.Dir
	}
	stdinPath := filepath.Join(hostWorkDir, ".flowbot-stdin")
	if wr := e.host.WriteFile(ctx, stdinPath, opts.Stdin, 0o644); !wr.IsOk() {
		ferr := wr.ErrorValue()
		err := result.NewExecutionError("spawn_error", ferr.Error(), nil)
		return &err
	}
	if err := ensureSandboxAgentReadable(stdinPath); err != nil {
		execErr := result.NewExecutionError("spawn_error", err.Error(), err)
		return &execErr
	}
	stdinRedirect := ".flowbot-stdin"
	if runOpts.WorkspaceInject {
		stdinRedirect = shellQuote(containerInjectWorkDir + "/.flowbot-stdin")
	}
	if len(opts.Argv) > 0 {
		runOpts.Command = shellJoin(opts.Argv) + " < " + stdinRedirect
		runOpts.Argv = nil
	} else {
		runOpts.Command = "(" + opts.Command + ") < " + stdinRedirect
	}
	return nil
}

func classifyExecError(runCtx context.Context, err error) (string, error) {
	cause := err
	switch {
	case runCtx.Err() == context.DeadlineExceeded || errors.Is(err, context.DeadlineExceeded):
		if runCtx.Err() != nil {
			cause = runCtx.Err()
		}
		return "timeout", cause
	case runCtx.Err() == context.Canceled || errors.Is(err, context.Canceled):
		if runCtx.Err() != nil {
			cause = runCtx.Err()
		}
		return "aborted", cause
	default:
		return "spawn_error", cause
	}
}

// DockerRunner runs commands via the Docker Engine API.
type DockerRunner struct{}

// Run starts an ephemeral container, waits for exit, and returns captured output.
func (DockerRunner) Run(ctx context.Context, opts RunOptions) (env.Capture, error) {
	if err := validateRunOptions(opts); err != nil {
		return env.Capture{}, err
	}

	var cleanupDirs []string
	defer func() {
		for _, d := range cleanupDirs {
			_ = os.RemoveAll(d)
		}
	}()
	if opts.AccessToken != "" && opts.CLIConfigDir == "" {
		dir, matErr := materializeCLIConfig(opts.ServerURL, opts.AccessToken)
		if matErr != nil {
			return env.Capture{}, matErr
		}
		opts.CLIConfigDir = dir
		cleanupDirs = append(cleanupDirs, dir)
	}
	if dir := injectCLIBinary(&opts); dir != "" {
		cleanupDirs = append(cleanupDirs, dir)
	}

	cmd, err := buildCommand(opts)
	if err != nil {
		return env.Capture{}, err
	}

	cli, err := client.New(client.FromEnv)
	if err != nil {
		return env.Capture{}, err
	}
	defer cli.Close()

	hostConfig, err := buildHostConfig(opts)
	if err != nil {
		return env.Capture{}, err
	}
	workDir := opts.WorkDir
	if workDir == "" {
		workDir = containerWorkspacePath(opts.Workspace)
	}
	flog.Info("[sandbox] container create image=%s workspace=%s workdir=%s cli_creds=%s %s",
		opts.Image, opts.Workspace, workDir, cliCredsLabel(opts.AccessToken), summarizeCommand(opts))
	// Create must not be cancelled mid-flight or Docker can leave Status=created orphans
	// while the client returns an error without an ID to clean up.
	createCtx, createCancel := context.WithTimeout(context.WithoutCancel(ctx), defaultCreateWait)
	defer createCancel()
	resp, err := cli.ContainerCreate(createCtx, client.ContainerCreateOptions{
		Config:     buildContainerConfig(opts, cmd, workDir),
		HostConfig: hostConfig,
	})
	if err != nil {
		flog.Info("[sandbox] container create failed image=%s workspace=%s err=%s",
			opts.Image, opts.Workspace, err.Error())
		return env.Capture{}, err
	}
	id := resp.ID
	flog.Info("[sandbox] container created id=%s image=%s workdir=%s", id, opts.Image, workDir)
	defer removeSandboxContainer(cli, id)
	if err := ctx.Err(); err != nil {
		return env.Capture{}, err
	}
	if opts.WorkspaceInject {
		if err := injectWorkspace(ctx, cli, id, opts.Workspace); err != nil {
			return env.Capture{}, err
		}
	}
	return waitAndCollectLogs(ctx, cli, id, opts.Workspace, workDir)
}

// buildContainerConfig builds the container config for a sandbox exec.
func buildContainerConfig(opts RunOptions, cmd []string, workDir string) *container.Config {
	return &container.Config{
		Image:      opts.Image,
		Cmd:        cmd,
		WorkingDir: workDir,
		Env:        buildContainerEnv(opts),
		Tty:        false,
		Labels: map[string]string{
			labelComponentKey: labelComponentSandbox,
		},
	}
}

// removeSandboxContainer force-removes a sandbox container using a detached timeout context.
func removeSandboxContainer(cli *client.Client, id string) {
	rctx, cancel := context.WithTimeout(context.Background(), defaultStopWait)
	defer cancel()
	if _, err := cli.ContainerRemove(rctx, id, client.ContainerRemoveOptions{Force: true, RemoveVolumes: true}); err != nil {
		if errdefs.IsNotFound(err) {
			return
		}
		flog.Warn("[sandbox] container remove failed id=%s err=%s", id, err.Error())
	}
}

func validateRunOptions(opts RunOptions) error {
	if strings.TrimSpace(opts.Workspace) == "" {
		return fmt.Errorf("sandbox: workspace is required")
	}
	if strings.TrimSpace(opts.Image) == "" {
		return fmt.Errorf("sandbox: image is required")
	}
	return nil
}

func buildHostConfig(opts RunOptions) (*container.HostConfig, error) {
	hostConfig := &container.HostConfig{}
	if !opts.WorkspaceInject {
		containerPath := containerWorkspacePath(opts.Workspace)
		hostConfig.Binds = append(hostConfig.Binds, fmt.Sprintf("%s:%s", opts.Workspace, containerPath))
	}
	if opts.CLIConfigDir != "" {
		hostConfig.Binds = append(hostConfig.Binds,
			fmt.Sprintf("%s:%s:ro", opts.CLIConfigDir, containerCLIConfigPath))
	}
	if opts.CLIBinaryDir != "" {
		hostConfig.Binds = append(hostConfig.Binds,
			fmt.Sprintf("%s:%s:ro", opts.CLIBinaryDir, containerCLIDirPath))
	}
	if opts.Network != "" {
		hostConfig.NetworkMode = container.NetworkMode(opts.Network)
	}
	if opts.Memory != "" {
		n, parseErr := units.RAMInBytes(opts.Memory)
		if parseErr != nil {
			return nil, fmt.Errorf("sandbox: memory: %w", parseErr)
		}
		hostConfig.Resources.Memory = n
	}
	if needsHostGateway(opts.ServerURL) {
		hostConfig.ExtraHosts = []string{hostDockerInternal + ":host-gateway"}
	}
	return hostConfig, nil
}

func injectWorkspace(ctx context.Context, cli *client.Client, containerID, hostWorkspace string) error {
	destName := strings.TrimPrefix(containerInjectWorkDir, "/")
	tarStream, err := tarWorkspace(hostWorkspace, destName)
	if err != nil {
		return fmt.Errorf("sandbox: tar workspace: %w", err)
	}
	flog.Info("[sandbox] workspace inject id=%s host=%s dest=%s", containerID, hostWorkspace, containerInjectWorkDir)
	if _, err := cli.CopyToContainer(ctx, containerID, client.CopyToContainerOptions{
		DestinationPath: "/",
		Content:         tarStream,
	}); err != nil {
		return fmt.Errorf("sandbox: copy workspace into container: %w", err)
	}
	return nil
}

func tarWorkspace(root, destName string) (io.Reader, error) {
	root = filepath.Clean(root)
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("workspace is not a directory")
	}
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	dirHdr := &tar.Header{
		Typeflag: tar.TypeDir,
		Name:     destName + "/",
		Mode:     0755,
		Uid:      sandboxAgentUID,
		Gid:      sandboxAgentGID,
	}
	if err := tw.WriteHeader(dirHdr); err != nil {
		_ = tw.Close()
		return nil, err
	}
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		return writeWorkspaceTarEntry(tw, root, destName, path, d, walkErr)
	})
	if err != nil {
		_ = tw.Close()
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	return &buf, nil
}

func writeWorkspaceTarEntry(tw *tar.Writer, root, destName, path string, d fs.DirEntry, walkErr error) error {
	if walkErr != nil {
		return walkErr
	}
	rel, relErr := filepath.Rel(root, path)
	if relErr != nil {
		return relErr
	}
	if rel == "." {
		return nil
	}
	fi, statErr := d.Info()
	if statErr != nil {
		return statErr
	}
	hdr, hdrErr := tar.FileInfoHeader(fi, "")
	if hdrErr != nil {
		return hdrErr
	}
	hdr.Name = destName + "/" + filepath.ToSlash(rel)
	hdr.Uid = sandboxAgentUID
	hdr.Gid = sandboxAgentGID
	if d.IsDir() {
		hdr.Name = strings.TrimSuffix(hdr.Name, "/") + "/"
		hdr.Typeflag = tar.TypeDir
		hdr.Mode = int64(hdr.Mode&0777 | 0755)
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	if d.IsDir() || !fi.Mode().IsRegular() {
		return nil
	}
	return copyFileToTar(tw, path)
}

func copyFileToTar(tw *tar.Writer, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(tw, f)
	closeErr := f.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

// materializeCLIConfig writes CLI token/server_url files into a temporary host directory
// outside the agent workspace so tools cannot read credentials from the workspace mount.
// Files are made readable by the sandbox agent user (uid/gid 1000).
func materializeCLIConfig(serverURL, token string) (string, error) {
	dir, err := os.MkdirTemp("", cliConfigTempDirPattern)
	if err != nil {
		return "", fmt.Errorf("sandbox: create cli config dir: %w", err)
	}
	cleanupOnErr := true
	defer func() {
		if cleanupOnErr {
			_ = os.RemoveAll(dir)
		}
	}()
	if token != "" {
		path := filepath.Join(dir, cliTokenFileName)
		if err := os.WriteFile(path, []byte(token), cliConfigOwnerOnly); err != nil {
			return "", fmt.Errorf("sandbox: write token: %w", err)
		}
		if err := ensureSandboxAgentReadable(path); err != nil {
			return "", err
		}
	}
	if serverURL != "" {
		path := filepath.Join(dir, cliServerURLFileName)
		if err := os.WriteFile(path, []byte(serverURL), cliConfigOwnerOnly); err != nil {
			return "", fmt.Errorf("sandbox: write server_url: %w", err)
		}
		if err := ensureSandboxAgentReadable(path); err != nil {
			return "", err
		}
	}
	if err := ensureSandboxAgentReadable(dir); err != nil {
		return "", err
	}
	cleanupOnErr = false
	return dir, nil
}

// injectCLIBinary stages opts.CLIBinary into a directory bind when CLIBinaryDir is empty.
// Staging failure warns once and leaves CLIBinaryDir empty (shell/code still run).
func injectCLIBinary(opts *RunOptions) string {
	if opts.CLIBinary == "" || opts.CLIBinaryDir != "" {
		return ""
	}
	dir, err := materializeCLIBinaryDir(opts.CLIBinary)
	if err != nil {
		warnMissingCLI("stage cli binary: %s", err.Error())
		return ""
	}
	opts.CLIBinaryDir = dir
	return dir
}

// materializeCLIBinaryDir copies src as "flowbot" into a temp dir beside src
// (fallback: default temp) so a Docker daemon that can see the CLI file can see the stage dir.
func materializeCLIBinaryDir(src string) (string, error) {
	dir, err := os.MkdirTemp(filepath.Dir(src), cliBinaryTempDirPattern)
	if err != nil {
		dir, err = os.MkdirTemp("", cliBinaryTempDirPattern)
	}
	if err != nil {
		return "", fmt.Errorf("sandbox: create cli binary dir: %w", err)
	}
	cleanupOnErr := true
	defer func() {
		if cleanupOnErr {
			_ = os.RemoveAll(dir)
		}
	}()
	dest := filepath.Join(dir, containerCLIName)
	if err := copyFile(src, dest); err != nil {
		return "", fmt.Errorf("sandbox: copy cli binary: %w", err)
	}
	if err := ensureSandboxAgentExecutable(dest); err != nil {
		return "", err
	}
	if err := ensureSandboxAgentExecutable(dir); err != nil {
		return "", err
	}
	cleanupOnErr = false
	return dir, nil
}

func copyFile(src, dest string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := in.Close(); err == nil {
			err = cerr
		}
	}()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, cliExecWorld)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := out.Close(); err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return out.Chmod(cliExecWorld)
}

func chownOrWorldChmod(path string, ownerOnly, world os.FileMode, what string) error {
	if err := os.Chown(path, sandboxAgentUID, sandboxAgentGID); err != nil {
		if chmodErr := os.Chmod(path, world); chmodErr != nil {
			return fmt.Errorf("sandbox: make %s accessible (chown: %v; chmod: %w)", what, err, chmodErr)
		}
		return nil
	}
	if err := os.Chmod(path, ownerOnly); err != nil {
		return fmt.Errorf("sandbox: chmod %s: %w", what, err)
	}
	return nil
}

func ensureSandboxAgentExecutable(path string) error {
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("sandbox: stat cli binary: %w", err)
	}
	return chownOrWorldChmod(path, cliExecOwner, cliExecWorld, "cli binary")
}

// EnsureAgentReadable makes path readable by the sandbox container user (uid 1000).
// Prefer chown to the agent user with owner-only mode; if chown fails (non-root / Windows),
// fall back to world-accessible mode for the ephemeral temp path.
// Directories use modes with the execute bit so the agent can traverse into them.
func EnsureAgentReadable(path string) error {
	return ensureSandboxAgentReadable(path)
}

func ensureSandboxAgentReadable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("sandbox: stat cli config: %w", err)
	}
	ownerOnly := os.FileMode(cliConfigOwnerOnly)
	worldAccessible := os.FileMode(cliConfigWorldReadable)
	if info.IsDir() {
		ownerOnly = cliConfigDirOwnerOnly
		worldAccessible = cliConfigDirWorldAccessible
	}
	return chownOrWorldChmod(path, ownerOnly, worldAccessible, "cli config")
}

// buildContainerEnv returns Docker Env entries for the flowbot CLI and caller-supplied vars.
func buildContainerEnv(opts RunOptions) []string {
	var out []string
	if opts.AccessToken != "" {
		if opts.ServerURL != "" {
			out = append(out, envFlowbotServerURL+"="+opts.ServerURL)
		}
		out = append(out, envFlowbotToken+"="+opts.AccessToken)
	}
	if len(opts.Env) > 0 {
		out = append(out, opts.Env...)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// needsHostGateway reports whether ExtraHosts should map host.docker.internal to the host gateway.
func needsHostGateway(serverURL string) bool {
	serverURL = strings.TrimSpace(serverURL)
	if serverURL == "" {
		return false
	}
	u, err := url.Parse(serverURL)
	if err != nil {
		return strings.Contains(serverURL, hostDockerInternal)
	}
	host := u.Hostname()
	if host == "" {
		return strings.Contains(serverURL, hostDockerInternal)
	}
	return host == hostDockerInternal
}

func cliCredsLabel(accessToken string) string {
	if accessToken != "" {
		return "injected"
	}
	return "none"
}

func waitAndCollectLogs(ctx context.Context, cli *client.Client, id, workspace, workDir string) (env.Capture, error) {
	if _, err := cli.ContainerStart(ctx, id, client.ContainerStartOptions{}); err != nil {
		return env.Capture{}, err
	}
	wait := cli.ContainerWait(ctx, id, client.ContainerWaitOptions{Condition: container.WaitConditionNotRunning})
	var exitCode int64
	select {
	case err := <-wait.Error:
		if err != nil {
			return env.Capture{}, err
		}
	case status := <-wait.Result:
		exitCode = status.StatusCode
	case <-ctx.Done():
		// Caller defer force-removes; stop first so a running container can exit cleanly.
		stopCtx, cancel := context.WithTimeout(context.Background(), defaultStopWait)
		defer cancel()
		_, _ = cli.ContainerStop(stopCtx, id, client.ContainerStopOptions{})
		return env.Capture{}, ctx.Err()
	}
	logs, err := cli.ContainerLogs(ctx, id, client.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return env.Capture{}, err
	}
	defer logs.Close()
	output, err := io.ReadAll(logs)
	if err != nil {
		return env.Capture{}, err
	}
	stdout, stderr := demuxDockerLogs(output)
	flog.Info("[sandbox] container done id=%s workspace=%s workdir=%s exit_code=%d",
		id, workspace, workDir, exitCode)
	return env.Capture{Stdout: stdout, Stderr: stderr, ExitCode: int(exitCode)}, nil
}

// containerWorkspacePath normalizes the workspace path for container bind mounts and WorkingDir.
func containerWorkspacePath(workspace string) string {
	return strings.ReplaceAll(filepath.Clean(workspace), "\\", "/")
}

func summarizeCommand(opts RunOptions) string {
	if len(opts.Argv) > 0 {
		return "argv=" + truncateForLog(strings.Join(opts.Argv, " "))
	}
	return "command=" + truncateForLog(strings.TrimSpace(opts.Command))
}

func truncateForLog(text string) string {
	if len(text) <= maxLoggedCommandLen {
		return text
	}
	return text[:maxLoggedCommandLen] + "..."
}

func buildCommand(opts RunOptions) ([]string, error) {
	var cmd []string
	switch {
	case len(opts.Argv) > 0:
		cmd = append([]string(nil), opts.Argv...)
	case strings.TrimSpace(opts.Command) == "":
		return nil, fmt.Errorf("sandbox: empty command")
	default:
		cmd = []string{"sh", "-c", opts.Command}
	}
	if opts.CLIBinaryDir != "" {
		return wrapCommandWithCLIPath(cmd), nil
	}
	return cmd, nil
}

func wrapCommandWithCLIPath(cmd []string) []string {
	script := `PATH="` + containerCLIDirPath + `:$PATH" exec "$0" "$@"`
	return append([]string{"sh", "-c", script, cmd[0]}, cmd[1:]...)
}

func stripDockerLogHeaders(data []byte) string {
	stdout, stderr := demuxDockerLogs(data)
	if stderr == "" {
		return stdout
	}
	if stdout == "" {
		return stderr
	}
	return stdout + stderr
}

// demuxDockerLogs splits Docker multiplexed log frames (stream byte 1=stdout, 2=stderr).
func demuxDockerLogs(data []byte) (stdout, stderr string) {
	if len(data) < 8 {
		return string(data), ""
	}
	var outBuf, errBuf bytes.Buffer
	rest := data
	wrote := false
	for len(rest) >= 8 {
		stream := rest[0]
		size := int(rest[4])<<24 | int(rest[5])<<16 | int(rest[6])<<8 | int(rest[7])
		rest = rest[8:]
		if size > len(rest) {
			chunk := rest
			switch stream {
			case 2:
				_, _ = errBuf.Write(chunk)
			default:
				_, _ = outBuf.Write(chunk)
			}
			wrote = true
			break
		}
		chunk := rest[:size]
		rest = rest[size:]
		switch stream {
		case 2:
			_, _ = errBuf.Write(chunk)
		default:
			_, _ = outBuf.Write(chunk)
		}
		wrote = true
	}
	if !wrote {
		return string(data), ""
	}
	return outBuf.String(), errBuf.String()
}

func shellJoin(argv []string) string {
	parts := make([]string, len(argv))
	for i, arg := range argv {
		parts[i] = shellQuote(arg)
	}
	return strings.Join(parts, " ")
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

// Ensure Env implements ExecutionEnv.
var _ env.ExecutionEnv = (*Env)(nil)
