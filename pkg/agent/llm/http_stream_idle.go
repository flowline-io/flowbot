package llm

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/flowline-io/flowbot/pkg/flog"
)

// streamIdleTransport wraps streaming chat completion response bodies with read idle timeouts.
type streamIdleTransport struct {
	base http.RoundTripper
}

func (t *streamIdleTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	resp, err := base.RoundTrip(req)
	if err != nil || resp == nil || resp.Body == nil {
		return resp, err
	}
	if !isChatCompletionsPath(req.URL.Path) {
		return resp, err
	}
	idle := streamIdleTimeout()
	if idle <= 0 {
		return resp, err
	}
	resp.Body = newIdleTimeoutReadCloser(resp.Body, idle)
	return resp, err
}

func isChatCompletionsPath(path string) bool {
	return strings.Contains(path, "chat/completions")
}

type idleTimeoutReadCloser struct {
	rc          io.ReadCloser
	idleTimeout time.Duration
	mu          sync.Mutex
	lastRead    time.Time
	closed      bool
}

func newIdleTimeoutReadCloser(rc io.ReadCloser, idleTimeout time.Duration) *idleTimeoutReadCloser {
	return &idleTimeoutReadCloser{
		rc:          rc,
		idleTimeout: idleTimeout,
		lastRead:    time.Now(),
	}
}

func (r *idleTimeoutReadCloser) Read(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return 0, fmt.Errorf("http stream read idle timeout: %w", ErrStreamIdle)
	}

	wait := r.waitDurationLocked()
	if wait <= 0 {
		return 0, r.closeWithIdleErrorLocked()
	}

	// Read into an owned buffer so a timed-out goroutine cannot race on caller's p.
	buf := make([]byte, len(p))
	type readResult struct {
		n   int
		err error
	}
	ch := make(chan readResult, 1)
	go func() {
		n, err := r.rc.Read(buf)
		ch <- readResult{n: n, err: err}
	}()

	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case res := <-ch:
		if res.n > 0 {
			copy(p, buf[:res.n])
			r.lastRead = time.Now()
		}
		return res.n, res.err
	case <-timer.C:
		return 0, r.closeWithIdleErrorLocked()
	}
}

func (r *idleTimeoutReadCloser) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	return r.rc.Close()
}

func (r *idleTimeoutReadCloser) waitDurationLocked() time.Duration {
	remaining := r.idleTimeout - time.Since(r.lastRead)
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (r *idleTimeoutReadCloser) closeWithIdleErrorLocked() error {
	r.closed = true
	_ = r.rc.Close()
	flog.Warn("[agent-llm] http stream read idle timeout limit=%s", r.idleTimeout)
	return fmt.Errorf("http stream read idle timeout: %w", ErrStreamIdle)
}

// StreamIdleTransportForTest exposes the stream idle transport for tests.
func StreamIdleTransportForTest(base http.RoundTripper) http.RoundTripper {
	return &streamIdleTransport{base: base}
}

// IdleTimeoutReadCloserForTest exposes the idle timeout reader for tests.
func IdleTimeoutReadCloserForTest(rc io.ReadCloser, idleTimeout time.Duration) io.ReadCloser {
	return newIdleTimeoutReadCloser(rc, idleTimeout)
}

// IsStreamIdleError reports whether an error was caused by a stalled LLM response body read.
func IsStreamIdleError(err error) bool {
	return errors.Is(err, ErrStreamIdle)
}
