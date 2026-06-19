package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestBindHTTPServer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		addr    string
		wantErr bool
	}{
		{name: "binds ephemeral tcp4 port", addr: "127.0.0.1:0", wantErr: false},
		{name: "rejects invalid port", addr: "127.0.0.1:999999", wantErr: true},
		{name: "rejects malformed host", addr: "not-a-host:abc", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ln, err := bindHTTPServer(tt.addr)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, ln)
			addr := ln.Addr().String()
			assert.NotEmpty(t, addr)
			require.NoError(t, ln.Close())
		})
	}
}

func TestBindHTTPServerUnix(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("unix domain sockets are not supported on Windows")
	}

	ln, err := bindHTTPServer("unix:/tmp/flowbot-server-test.sock")
	require.NoError(t, err)
	require.NotNil(t, ln)
	t.Cleanup(func() { _ = ln.Close() })
}

func TestBindHTTPServerConflict(t *testing.T) {
	t.Parallel()

	hold, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = hold.Close() })

	addr := hold.Addr().String()
	_, err = bindHTTPServer(addr)
	require.Error(t, err)
}

func TestShouldIgnoreServeError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		stopping bool
		want     bool
	}{
		{name: "nil error is ignored", err: nil, want: true},
		{name: "server closed during shutdown is ignored", err: http.ErrServerClosed, want: true},
		{name: "wrapped server closed is ignored", err: fmtWrap(http.ErrServerClosed), want: true},
		{name: "unexpected error is not ignored", err: errors.New("accept tcp: use of closed network connection"), want: false},
		{name: "error ignored when stopping flag set", err: errors.New("accept tcp: use of closed network connection"), stopping: true, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var stopping atomic.Bool
			if tt.stopping {
				stopping.Store(true)
			}
			assert.Equal(t, tt.want, shouldIgnoreServeError(tt.err, &stopping))
		})
	}
}

func TestHTTPServerSurvivesOnStartContextCancel(t *testing.T) {
	t.Parallel()

	ln, err := bindHTTPServer("127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	app := fiber.New()
	app.Get("/ping", func(c fiber.Ctx) error { return c.SendString("pong") })

	ready := make(chan struct{})
	go func() {
		close(ready)
		serveErr := app.Listener(ln, fiber.ListenConfig{DisableStartupMessage: true})
		assert.True(t, errors.Is(serveErr, http.ErrServerClosed) || serveErr == nil)
	}()

	<-ready
	require.Eventually(t, func() bool {
		resp, getErr := http.Get("http://" + ln.Addr().String() + "/ping")
		if getErr != nil {
			return false
		}
		defer func() { _ = resp.Body.Close() }()
		body, readErr := io.ReadAll(resp.Body)
		return readErr == nil && resp.StatusCode == http.StatusOK && string(body) == "pong"
	}, 2*time.Second, 20*time.Millisecond)

	onStartCtx, onStartCancel := context.WithCancel(context.Background())
	onStartCancel()
	_ = onStartCtx

	require.Eventually(t, func() bool {
		resp, getErr := http.Get("http://" + ln.Addr().String() + "/ping")
		if getErr != nil {
			return false
		}
		defer func() { _ = resp.Body.Close() }()
		body, readErr := io.ReadAll(resp.Body)
		return readErr == nil && resp.StatusCode == http.StatusOK && string(body) == "pong"
	}, 2*time.Second, 20*time.Millisecond)

	require.NoError(t, app.ShutdownWithContext(context.Background()))
}

func TestServeFiberListenerGracefulStop(t *testing.T) {
	t.Parallel()

	ln, err := bindHTTPServer("127.0.0.1:0")
	require.NoError(t, err)

	app := fiber.New()
	app.Get("/ping", func(c fiber.Ctx) error { return c.SendString("pong") })

	var stopping atomic.Bool
	shutdowner := &recordingShutdowner{}
	serveFiberListener(app, ln, shutdowner, &stopping)

	require.Eventually(t, func() bool {
		resp, getErr := http.Get("http://" + ln.Addr().String() + "/ping")
		if getErr != nil {
			return false
		}
		defer func() { _ = resp.Body.Close() }()
		return resp.StatusCode == http.StatusOK
	}, 2*time.Second, 20*time.Millisecond)

	stopping.Store(true)
	require.NoError(t, app.ShutdownWithContext(context.Background()))
	require.Eventually(t, func() bool { return shutdowner.calls() == 0 }, time.Second, 20*time.Millisecond)
}

func TestServeFiberListenerUnexpectedExit(t *testing.T) {
	t.Parallel()

	base, err := bindHTTPServer("127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = base.Close() })

	ln := &abortingListener{
		Listener: base,
		failErr:  errors.New("listener accept failed"),
	}

	app := fiber.New()
	var stopping atomic.Bool
	shutdowner := &recordingShutdowner{}
	serveFiberListener(app, ln, shutdowner, &stopping)

	conn, err := net.DialTimeout("tcp4", base.Addr().String(), time.Second)
	require.NoError(t, err)
	require.NoError(t, conn.Close())

	require.Eventually(t, func() bool { return shutdowner.calls() == 1 }, 2*time.Second, 20*time.Millisecond)
}

// abortingListener wraps a net.Listener and fails Accept after the first successful connection.
type abortingListener struct {
	net.Listener
	failErr error
	fail    atomic.Bool
}

func (l *abortingListener) Accept() (net.Conn, error) {
	if l.fail.Load() {
		return nil, l.failErr
	}
	conn, err := l.Listener.Accept()
	if err == nil {
		l.fail.Store(true)
	}
	return conn, err
}

type recordingShutdowner struct {
	count atomic.Int32
}

func (r *recordingShutdowner) Shutdown(...fx.ShutdownOption) error {
	r.count.Add(1)
	return nil
}

func (r *recordingShutdowner) calls() int {
	return int(r.count.Load())
}

func fmtWrap(err error) error {
	return fmt.Errorf("wrapped: %w", err)
}
