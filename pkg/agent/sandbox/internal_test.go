package sandbox

import (
	"archive/tar"
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/flowline-io/flowbot/pkg/agent/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTarWorkspace(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".flowbot-stdin"), []byte(`{}`), 0o644))
	sub := filepath.Join(root, "sub")
	require.NoError(t, os.Mkdir(sub, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(sub, "x.txt"), []byte("x"), 0o644))

	r, err := tarWorkspace(root, "workspace")
	require.NoError(t, err)
	tr := tar.NewReader(r)
	names := map[string]bool{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		names[hdr.Name] = true
		assert.Equal(t, sandboxAgentUID, hdr.Uid)
		assert.Equal(t, sandboxAgentGID, hdr.Gid)
	}
	assert.True(t, names["workspace/"] || names["workspace"])
	assert.True(t, names["workspace/main.go"])
	assert.True(t, names["workspace/.flowbot-stdin"])
	assert.True(t, names["workspace/sub/x.txt"])
}

func TestRemapHostPathsInEnv(t *testing.T) {
	t.Parallel()
	got := remapHostPathsInEnv([]string{
		"GOCACHE=/tmp/ws/.gocache",
		"GOPATH=/tmp/ws/.gopath",
		"OTHER=/elsewhere",
		"BARE",
	}, "/tmp/ws", "/workspace")
	assert.Equal(t, []string{
		"GOCACHE=/workspace/.gocache",
		"GOPATH=/workspace/.gopath",
		"OTHER=/elsewhere",
		"BARE",
	}, got)
}

func TestValidateRunOptions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		opts    RunOptions
		wantErr string
	}{
		{name: "valid options", opts: RunOptions{Workspace: "/ws", Image: "img:1"}},
		{name: "missing workspace", opts: RunOptions{Image: "img:1"}, wantErr: "workspace is required"},
		{name: "missing image", opts: RunOptions{Workspace: "/ws"}, wantErr: "image is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateRunOptions(tt.opts)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestBuildCommand(t *testing.T) {
	t.Parallel()
	pathWrap := []string{"sh", "-c", `PATH="/opt/flowbot-cli:$PATH" exec "$0" "$@"`}
	tests := []struct {
		name    string
		opts    RunOptions
		want    []string
		wantErr string
	}{
		{name: "argv always wraps path", opts: RunOptions{Argv: []string{"python", "main.py"}}, want: append(append([]string{}, pathWrap...), "python", "main.py")},
		{name: "shell wraps command", opts: RunOptions{Command: "echo hi"}, want: append(append([]string{}, pathWrap...), "sh", "-c", "echo hi")},
		{name: "empty command errors", opts: RunOptions{}, wantErr: "empty command"},
		{
			name: "cli dir still wraps path on argv",
			opts: RunOptions{Argv: []string{"python", "main.py"}, CLIBinaryDir: "/tmp/cli"},
			want: append(append([]string{}, pathWrap...), "python", "main.py"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := buildCommand(tt.opts)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStripDockerLogHeaders(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   []byte
		want string
	}{
		{name: "short payload unchanged", in: []byte("hi"), want: "hi"},
		{name: "empty payload", in: []byte{}, want: ""},
		{name: "framed docker logs stripped", in: append([]byte{1, 0, 0, 0, 0, 0, 0, 3}, []byte("abc")...), want: "abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, stripDockerLogHeaders(tt.in))
		})
	}
}

func TestBuildHostConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		opts           RunOptions
		wantErr        string
		wantBinds      int
		wantExtraHosts bool
	}{
		{name: "binds workspace", opts: RunOptions{Workspace: "/host/ws"}, wantBinds: 1},
		{name: "inject skips workspace bind", opts: RunOptions{Workspace: "/host/ws", WorkspaceInject: true}, wantBinds: 0},
		{name: "sets network mode", opts: RunOptions{Workspace: "/host/ws", Network: "bridge"}, wantBinds: 1},
		{name: "invalid memory", opts: RunOptions{Workspace: "/host/ws", Memory: "not-memory"}, wantErr: "memory"},
		{
			name: "host gateway without cli binds",
			opts: RunOptions{
				Workspace:    "/host/ws",
				CLIConfigDir: "/tmp/cli-cfg",
				CLIBinaryDir: "/tmp/cli-bin",
				ServerURL:    "http://host.docker.internal:6060",
			},
			wantBinds:      1,
			wantExtraHosts: true,
		},
		{
			name: "inject keeps no cli binds",
			opts: RunOptions{
				Workspace:       "/host/ws",
				WorkspaceInject: true,
				CLIConfigDir:    "/tmp/cli-cfg",
				CLIBinaryDir:    "/tmp/cli-bin",
			},
			wantBinds: 0,
		},
		{
			name: "no host gateway for other urls",
			opts: RunOptions{
				Workspace: "/host/ws",
				ServerURL: "http://flowbot:6060",
			},
			wantBinds:      1,
			wantExtraHosts: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			hc, err := buildHostConfig(tt.opts)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Len(t, hc.Binds, tt.wantBinds)
			if tt.wantExtraHosts {
				require.Equal(t, []string{"host.docker.internal:host-gateway"}, hc.ExtraHosts)
			} else {
				assert.Empty(t, hc.ExtraHosts)
			}
			for _, b := range hc.Binds {
				assert.NotContains(t, b, containerCLIConfigPath)
				assert.NotContains(t, b, containerCLIDirPath)
			}
		})
	}
}

func TestResolveCLIBinary(t *testing.T) {
	// Serial: sibling cases override package-level executableDir.

	t.Run("sibling of executable", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, siblingCLIBinaryName)
		require.NoError(t, os.WriteFile(path, []byte("x"), 0o755))
		prev := executableDir
		executableDir = func() (string, error) { return dir, nil }
		t.Cleanup(func() { executableDir = prev })
		got := ResolvedCLIBinary()
		abs, err := filepath.Abs(path)
		require.NoError(t, err)
		assert.Equal(t, abs, got)
	})

	t.Run("missing sibling", func(t *testing.T) {
		dir := t.TempDir()
		prev := executableDir
		executableDir = func() (string, error) { return dir, nil }
		t.Cleanup(func() { executableDir = prev })
		got := ResolvedCLIBinary()
		assert.Empty(t, got)
	})
}

func TestCLIBinaryStagingCopiesFile(t *testing.T) {
	t.Parallel()
	srcDir := t.TempDir()
	src := filepath.Join(srcDir, "flowbot-cli_linux_amd64")
	require.NoError(t, os.WriteFile(src, []byte("#!/bin/sh\necho ok\n"), 0o755))
	srcBefore, err := os.Stat(src)
	require.NoError(t, err)

	dir, err := materializeCLIBinaryDir(src)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	assert.Equal(t, filepath.Dir(src), filepath.Dir(dir))
	dest := filepath.Join(dir, containerCLIName)
	data, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, "#!/bin/sh\necho ok\n", string(data))
	assertAgentExecutable(t, dest)
	assertAgentExecutable(t, dir)

	destInfo, err := os.Stat(dest)
	require.NoError(t, err)
	assert.False(t, os.SameFile(srcBefore, destInfo), "staging must copy, not hardlink")
	srcAfter, err := os.Stat(src)
	require.NoError(t, err)
	assert.Equal(t, srcBefore.Mode(), srcAfter.Mode())
}

func TestCLIBinaryStagingMissingSrc(t *testing.T) {
	t.Parallel()
	_, err := materializeCLIBinaryDir(filepath.Join(t.TempDir(), "missing"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "copy cli binary")
}

func TestCLIInjectDegradesWhenStagingFails(t *testing.T) {
	t.Parallel()
	opts := RunOptions{CLIBinary: filepath.Join(t.TempDir(), "missing")}
	dir := injectCLIBinary(&opts)
	assert.Empty(t, dir)
	assert.Empty(t, opts.CLIBinaryDir)
}

func TestCLIInjectStagesWhenBinaryExists(t *testing.T) {
	t.Parallel()
	src := filepath.Join(t.TempDir(), "flowbot-cli_linux_amd64")
	require.NoError(t, os.WriteFile(src, []byte("x"), 0o755))
	opts := RunOptions{CLIBinary: src}
	dir := injectCLIBinary(&opts)
	require.NotEmpty(t, dir)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	assert.Equal(t, dir, opts.CLIBinaryDir)
	assert.Equal(t, filepath.Dir(src), filepath.Dir(dir))
}

func TestCLIInjectSkipsWhenDirAlreadySet(t *testing.T) {
	t.Parallel()
	src := filepath.Join(t.TempDir(), "flowbot-cli_linux_amd64")
	require.NoError(t, os.WriteFile(src, []byte("x"), 0o755))
	opts := RunOptions{CLIBinary: src, CLIBinaryDir: "/already"}
	assert.Empty(t, injectCLIBinary(&opts))
	assert.Equal(t, "/already", opts.CLIBinaryDir)
}

func assertAgentExecutable(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	require.NoError(t, err)
	if runtime.GOOS == "windows" {
		return
	}
	perm := info.Mode().Perm()
	assert.True(t, perm == cliExecOwner || perm&0o111 != 0,
		"path %s mode %o should be executable/traversable by sandbox agent", path, perm)
}

func TestEnsureSandboxAgentExecutable(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "flowbot")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0o600))
	require.NoError(t, ensureSandboxAgentExecutable(path))
	assertAgentExecutable(t, path)
	require.NoError(t, ensureSandboxAgentExecutable(path))
	assertAgentExecutable(t, path)
}

func TestMaterializeCLIConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		serverURL string
		token     string
		wantURL   bool
		wantToken bool
	}{
		{name: "writes both files", serverURL: "http://host.docker.internal:6060", token: "tok", wantURL: true, wantToken: true},
		{name: "token only", serverURL: "", token: "tok", wantURL: false, wantToken: true},
		{name: "url only", serverURL: "http://flowbot:6060", token: "", wantURL: true, wantToken: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir, err := materializeCLIConfig(tt.serverURL, tt.token)
			require.NoError(t, err)
			t.Cleanup(func() { _ = os.RemoveAll(dir) })
			assertAgentReadable(t, dir)

			tokenPath := filepath.Join(dir, cliTokenFileName)
			urlPath := filepath.Join(dir, cliServerURLFileName)
			if tt.wantToken {
				data, readErr := os.ReadFile(tokenPath)
				require.NoError(t, readErr)
				assert.Equal(t, tt.token, string(data))
				assertAgentReadable(t, tokenPath)
			} else {
				_, readErr := os.Stat(tokenPath)
				require.Error(t, readErr)
			}
			if tt.wantURL {
				data, readErr := os.ReadFile(urlPath)
				require.NoError(t, readErr)
				assert.Equal(t, tt.serverURL, string(data))
				assertAgentReadable(t, urlPath)
			} else {
				_, readErr := os.Stat(urlPath)
				require.Error(t, readErr)
			}
		})
	}
}

func assertAgentReadable(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	require.NoError(t, err)
	perm := info.Mode().Perm()
	if info.IsDir() {
		// Directories need an execute bit to be traversable; owner-only or world-accessible.
		assert.True(t, perm == cliConfigDirOwnerOnly || perm&0o001 != 0 || perm&0o010 != 0,
			"dir %s mode %o should be traversable by sandbox agent", path, perm)
		return
	}
	// After ensureSandboxAgentReadable: either owner-only (chown succeeded) or world-readable fallback.
	assert.True(t, perm == cliConfigOwnerOnly || perm&0o004 != 0 || perm&0o040 != 0,
		"path %s mode %o should be readable by sandbox agent", path, perm)
}

func TestEnsureAgentReadableOpensRestrictiveWorkspace(t *testing.T) {
	t.Parallel()
	dir, err := os.MkdirTemp("", "flowbot-sandbox-ws-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	// Directories need owner-execute to be traversable; 0600 would break the fixture.
	require.NoError(t, os.Chmod(dir, 0o700)) // #nosec G302 -- dir mode, not a file

	require.NoError(t, EnsureAgentReadable(dir))
	info, err := os.Stat(dir)
	require.NoError(t, err)
	perm := info.Mode().Perm()
	assert.True(t,
		perm == cliConfigDirOwnerOnly || perm&0o001 != 0 || perm&0o010 != 0,
		"workspace dir %s mode %o should be traversable by sandbox agent", dir, perm,
	)
}

func TestEnsureSandboxAgentReadable(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "file becomes readable after ensure"},
		{name: "directory remains traversable after ensure"},
		{name: "idempotent second ensure"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := filepath.Join(dir, "secret")
			var nested string
			if strings.Contains(tt.name, "directory") {
				path = filepath.Join(dir, "subdir")
				require.NoError(t, os.Mkdir(path, 0o700))
				nested = filepath.Join(path, "nested")
				require.NoError(t, os.WriteFile(nested, []byte("x"), 0o600))
			} else {
				require.NoError(t, os.WriteFile(path, []byte("x"), 0o600))
			}
			require.NoError(t, ensureSandboxAgentReadable(path))
			assertAgentReadable(t, path)
			if nested != "" {
				data, readErr := os.ReadFile(nested)
				require.NoError(t, readErr, "directory must stay traversable after ensure")
				assert.Equal(t, "x", string(data))
			}
			if strings.Contains(tt.name, "idempotent") {
				require.NoError(t, ensureSandboxAgentReadable(path))
				assertAgentReadable(t, path)
			}
		})
	}
}

func TestBuildContainerConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		opts    RunOptions
		cmd     []string
		workDir string
	}{
		{name: "labels sandbox component", opts: RunOptions{Image: "img:1"}, cmd: []string{"true"}, workDir: "/ws"},
		{name: "preserves image", opts: RunOptions{Image: "custom:2"}, cmd: []string{"echo"}, workDir: "/tmp"},
		{name: "preserves workdir", opts: RunOptions{Image: "img:1"}, cmd: []string{"pwd"}, workDir: "/ws/sub"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := buildContainerConfig(tt.opts, tt.cmd, tt.workDir)
			require.NotNil(t, got)
			assert.Equal(t, tt.opts.Image, got.Image)
			assert.Equal(t, []string(tt.cmd), []string(got.Cmd))
			assert.Equal(t, tt.workDir, got.WorkingDir)
			assert.Equal(t, labelComponentSandbox, got.Labels[labelComponentKey])
		})
	}
}

func TestBuildContainerEnv(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		opts RunOptions
		want []string
	}{
		{name: "empty without token", opts: RunOptions{ServerURL: "http://x"}, want: nil},
		{
			name: "token and url",
			opts: RunOptions{ServerURL: "http://host.docker.internal:6060", AccessToken: "tok"},
			want: []string{"FLOWBOT_SERVER_URL=http://host.docker.internal:6060", "FLOWBOT_TOKEN=tok"},
		},
		{
			name: "token only",
			opts: RunOptions{AccessToken: "tok"},
			want: []string{"FLOWBOT_TOKEN=tok"},
		},
		{
			name: "cli dir does not set path env",
			opts: RunOptions{CLIBinaryDir: "/tmp/cli-bin"},
			want: nil,
		},
		{
			name: "cli dir and token",
			opts: RunOptions{CLIBinaryDir: "/tmp/cli-bin", AccessToken: "tok"},
			want: []string{"FLOWBOT_TOKEN=tok"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, buildContainerEnv(tt.opts))
		})
	}
}

func TestNeedsHostGateway(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{name: "host docker internal", url: "http://host.docker.internal:6060", want: true},
		{name: "service name", url: "http://flowbot:6060", want: false},
		{name: "empty", url: "", want: false},
		{name: "localhost", url: "http://127.0.0.1:6060", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, needsHostGateway(tt.url))
		})
	}
}

func TestEnvExecForwardsCLIBinary(t *testing.T) {
	// Serial: overrides executableDir.
	dir := t.TempDir()
	cliPath := filepath.Join(dir, siblingCLIBinaryName)
	require.NoError(t, os.WriteFile(cliPath, []byte("#!/bin/sh\n"), 0o755))
	prev := executableDir
	executableDir = func() (string, error) { return dir, nil }
	t.Cleanup(func() { executableDir = prev })

	runner := &recordingRunner{cap: env.Capture{ExitCode: 0}}
	e := New(Config{Image: "img", Workspace: "/ws"}, env.Default(), runner)
	got := e.Exec(context.Background(), env.ExecOptions{Command: "flowbot version"})
	require.True(t, got.IsOk())
	abs, err := filepath.Abs(cliPath)
	require.NoError(t, err)
	assert.Equal(t, abs, runner.last.CLIBinary)
}

func TestEnvExecMissingCLIBinaryUsesEmptyPath(t *testing.T) {
	// Serial: overrides executableDir.
	dir := t.TempDir()
	prev := executableDir
	executableDir = func() (string, error) { return dir, nil }
	t.Cleanup(func() { executableDir = prev })

	runner := &recordingRunner{cap: env.Capture{ExitCode: 0}}
	e := New(Config{Image: "img", Workspace: "/ws"}, env.Default(), runner)
	got := e.Exec(context.Background(), env.ExecOptions{Command: "echo ok"})
	require.True(t, got.IsOk())
	assert.Empty(t, runner.last.CLIBinary)
}

type recordingRunner struct {
	last RunOptions
	cap  env.Capture
}

func (m *recordingRunner) Run(_ context.Context, opts RunOptions) (env.Capture, error) {
	m.last = opts
	return m.cap, nil
}

func TestTarFlowbotStub(t *testing.T) {
	t.Parallel()
	r, err := tarFlowbotStub()
	require.NoError(t, err)
	tr := tar.NewReader(r)
	var found bool
	for {
		hdr, nextErr := tr.Next()
		if nextErr == io.EOF {
			break
		}
		require.NoError(t, nextErr)
		if hdr.Name == "opt/flowbot-cli/flowbot" {
			found = true
			data, readErr := io.ReadAll(tr)
			require.NoError(t, readErr)
			assert.Contains(t, string(data), "CLI not available")
			assert.Equal(t, int64(cliExecWorld), hdr.Mode)
		}
	}
	assert.True(t, found)
}

func TestTarCLIConfigFiles(t *testing.T) {
	t.Parallel()
	r, err := tarCLIConfigFiles("http://host.docker.internal:6200", "tok")
	require.NoError(t, err)
	tr := tar.NewReader(r)
	names := map[string]string{}
	for {
		hdr, nextErr := tr.Next()
		if nextErr == io.EOF {
			break
		}
		require.NoError(t, nextErr)
		if hdr.Typeflag == tar.TypeReg {
			data, readErr := io.ReadAll(tr)
			require.NoError(t, readErr)
			names[hdr.Name] = string(data)
		}
	}
	assert.Equal(t, "tok", names["home/agent/.config/flowbot/token"])
	assert.Equal(t, "http://host.docker.internal:6200", names["home/agent/.config/flowbot/server_url"])
}

func TestEnsureKernCLIBinaryDirStub(t *testing.T) {
	t.Parallel()
	opts := RunOptions{}
	dir, err := ensureKernCLIBinaryDir(&opts)
	require.NoError(t, err)
	require.NotEmpty(t, dir)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	assert.Equal(t, dir, opts.CLIBinaryDir)
	data, err := os.ReadFile(filepath.Join(dir, containerCLIName))
	require.NoError(t, err)
	assert.Contains(t, string(data), "CLI not available")
}

func TestEnsureKernCLIBinaryDirReal(t *testing.T) {
	t.Parallel()
	src := filepath.Join(t.TempDir(), siblingCLIBinaryName)
	require.NoError(t, os.WriteFile(src, []byte("#!/bin/sh\necho ok\n"), 0o755))
	opts := RunOptions{CLIBinary: src}
	dir, err := ensureKernCLIBinaryDir(&opts)
	require.NoError(t, err)
	require.NotEmpty(t, dir)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	data, err := os.ReadFile(filepath.Join(dir, containerCLIName))
	require.NoError(t, err)
	assert.Equal(t, "#!/bin/sh\necho ok\n", string(data))
}
