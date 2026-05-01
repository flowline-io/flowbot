package homelab

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"golang.org/x/crypto/ssh"
)

type SSHRuntime struct {
	host     string
	port     int
	user     string
	password string
	key      string
}

func NewSSHRuntime(config RuntimeConfig) *SSHRuntime {
	port := config.SSHPort
	if port == 0 {
		port = 22
	}
	return &SSHRuntime{
		host:     config.SSHHost,
		port:     port,
		user:     config.SSHUser,
		password: config.SSHPassword,
		key:      config.SSHKey,
	}
}

func (r *SSHRuntime) clientConfig() (*ssh.ClientConfig, error) {
	config := &ssh.ClientConfig{
		User:            r.user,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	if r.key != "" {
		signer, err := ssh.ParsePrivateKey([]byte(r.key))
		if err != nil {
			return nil, fmt.Errorf("parse ssh key: %w", err)
		}
		config.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	} else if r.password != "" {
		config.Auth = []ssh.AuthMethod{ssh.Password(r.password)}
	} else {
		return nil, types.Errorf(types.ErrUnauthorized, "ssh requires key or password")
	}

	return config, nil
}

func (r *SSHRuntime) connect(ctx context.Context) (*ssh.Client, error) {
	cfg, err := r.clientConfig()
	if err != nil {
		return nil, err
	}

	addr := net.JoinHostPort(r.host, fmt.Sprintf("%d", r.port))

	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, types.WrapError(types.ErrUnavailable, "ssh dial", err)
	}

	c, chans, reqs, err := ssh.NewClientConn(conn, addr, cfg)
	if err != nil {
		_ = conn.Close()
		return nil, types.WrapError(types.ErrUnauthorized, "ssh handshake", err)
	}

	return ssh.NewClient(c, chans, reqs), nil
}

func (r *SSHRuntime) runRemote(ctx context.Context, app App, args ...string) (string, error) {
	client, err := r.connect(ctx)
	if err != nil {
		return "", err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", types.WrapError(types.ErrProvider, "ssh new session", err)
	}
	defer session.Close()

	composeFile := app.ComposeFile
	if composeFile == "" {
		composeFile = "docker-compose.yaml"
	}
	cmdArgs := append([]string{"compose", "-f", composeFile}, args...)
	cmdStr := "docker " + strings.Join(cmdArgs, " ")
	if app.Path != "" {
		cmdStr = "cd " + app.Path + " && " + cmdStr
	}

	output, err := session.CombinedOutput(cmdStr)
	if err != nil {
		return string(output), fmt.Errorf("%s: %w", strings.TrimSpace(string(output)), err)
	}
	return string(output), nil
}

func (r *SSHRuntime) Status(ctx context.Context, app App) (AppStatus, error) {
	if err := ctx.Err(); err != nil {
		return AppStatusUnknown, types.WrapError(types.ErrTimeout, "homelab status canceled", err)
	}

	output, err := r.runRemote(ctx, app, "ps", "--format", "json")
	if err != nil {
		flog.Warn("ssh docker compose ps failed for %s: %v", app.Name, err)
		return AppStatusUnknown, types.WrapError(types.ErrProvider, "docker compose ps via ssh", err)
	}

	return parseComposePSStatus(output), nil
}

func (r *SSHRuntime) Logs(ctx context.Context, app App, tail int) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "homelab logs canceled", err)
	}

	output, err := r.runRemote(ctx, app, "logs", fmt.Sprintf("--tail=%d", tail))
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "docker compose logs via ssh", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return []string{}, nil
	}
	return lines, nil
}

func (r *SSHRuntime) Start(ctx context.Context, app App) error {
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "homelab start canceled", err)
	}

	_, err := r.runRemote(ctx, app, "up", "-d")
	if err != nil {
		return types.WrapError(types.ErrProvider, "docker compose up via ssh", err)
	}
	return nil
}

func (r *SSHRuntime) Stop(ctx context.Context, app App) error {
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "homelab stop canceled", err)
	}

	_, err := r.runRemote(ctx, app, "down")
	if err != nil {
		return types.WrapError(types.ErrProvider, "docker compose down via ssh", err)
	}
	return nil
}

func (r *SSHRuntime) Restart(ctx context.Context, app App) error {
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "homelab restart canceled", err)
	}

	_, err := r.runRemote(ctx, app, "restart")
	if err != nil {
		return types.WrapError(types.ErrProvider, "docker compose restart via ssh", err)
	}
	return nil
}

func (r *SSHRuntime) Pull(ctx context.Context, app App) error {
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "homelab pull canceled", err)
	}

	_, err := r.runRemote(ctx, app, "pull")
	if err != nil {
		return types.WrapError(types.ErrProvider, "docker compose pull via ssh", err)
	}
	return nil
}

func (r *SSHRuntime) Update(ctx context.Context, app App) error {
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "homelab update canceled", err)
	}

	if _, err := r.runRemote(ctx, app, "pull"); err != nil {
		return types.WrapError(types.ErrProvider, "docker compose pull via ssh", err)
	}

	if _, err := r.runRemote(ctx, app, "up", "-d"); err != nil {
		return types.WrapError(types.ErrProvider, "docker compose up via ssh", err)
	}
	return nil
}
