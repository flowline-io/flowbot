package llm_test

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/agent/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type blockingReader struct {
	ch chan []byte
}

func (r *blockingReader) Read(p []byte) (int, error) {
	data, ok := <-r.ch
	if !ok {
		return 0, io.EOF
	}
	return copy(p, data), nil
}

func (*blockingReader) Close() error { return nil }

type blockingBodyRoundTripper struct {
	ch chan []byte
}

func (b *blockingBodyRoundTripper) RoundTrip(_ *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(&blockingReader{ch: b.ch}),
		Header:     make(http.Header),
	}, nil
}

func TestIdleTimeoutReadCloser(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() io.ReadCloser
		wantErrIdle bool
		wantRead    string
	}{
		{
			name: "returns data before idle timeout",
			setup: func() io.ReadCloser {
				ch := make(chan []byte, 1)
				ch <- []byte("ok")
				return llm.IdleTimeoutReadCloserForTest(&blockingReader{ch: ch}, 50*time.Millisecond)
			},
			wantRead: "ok",
		},
		{
			name: "times out when read blocks",
			setup: func() io.ReadCloser {
				ch := make(chan []byte)
				return llm.IdleTimeoutReadCloserForTest(&blockingReader{ch: ch}, 20*time.Millisecond)
			},
			wantErrIdle: true,
		},
		{
			name: "reads sequential chunks",
			setup: func() io.ReadCloser {
				ch := make(chan []byte, 2)
				ch <- []byte("a")
				ch <- []byte("b")
				return llm.IdleTimeoutReadCloserForTest(&blockingReader{ch: ch}, 50*time.Millisecond)
			},
			wantRead: "a",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := tt.setup()
			buf := make([]byte, 16)
			n, err := reader.Read(buf)
			if tt.wantErrIdle {
				require.Error(t, err)
				assert.True(t, llm.IsStreamIdleError(err))
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantRead, string(buf[:n]))
		})
	}
}

func TestStreamIdleTransportWrapsChatCompletions(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantIdle bool
	}{
		{name: "wraps chat completions", path: "/v1/chat/completions", wantIdle: true},
		{name: "wraps nested chat completions path", path: "/openai/v1/chat/completions", wantIdle: true},
		{name: "skips models path", path: "/v1/models", wantIdle: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var base http.RoundTripper
			if tt.wantIdle {
				ch := make(chan []byte)
				base = &blockingBodyRoundTripper{ch: ch}
			} else {
				base = roundTripFunc(func(_ *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader("")),
						Header:     make(http.Header),
					}, nil
				})
			}
			transport := llm.StreamIdleTransportForTest(base)
			req, err := http.NewRequest(http.MethodPost, "https://api.example.com"+tt.path, http.NoBody)
			require.NoError(t, err)

			resp, err := transport.RoundTrip(req)
			require.NoError(t, err)
			require.NotNil(t, resp.Body)
			defer func() { _ = resp.Body.Close() }()

			_, err = resp.Body.Read(make([]byte, 8))
			if tt.wantIdle {
				require.Error(t, err)
				assert.True(t, llm.IsStreamIdleError(err))
				return
			}
			require.ErrorIs(t, err, io.EOF)
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
