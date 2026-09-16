package browser

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssertNavigateURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		url          string
		allowPrivate bool
		allowHosts   []string
		wantErr      bool
	}{
		{name: "public https", url: "https://example.com/path", wantErr: false},
		{name: "private blocked", url: "http://10.0.0.1/", wantErr: true},
		{name: "private allowed", url: "http://10.0.0.1/", allowPrivate: true, wantErr: false},
		{name: "metadata blocked with allow private", url: "http://metadata.google.internal/", allowPrivate: true, wantErr: true},
		{name: "loopback blocked with allow private", url: "http://127.0.0.1/", allowPrivate: true, wantErr: true},
		{name: "link local blocked with allow private", url: "http://169.254.169.254/", allowPrivate: true, wantErr: true},
		{name: "host allowlist", url: "http://192.168.1.5/", allowHosts: []string{"192.168.1.5"}, wantErr: false},
		{name: "cidr allowlist", url: "http://192.168.1.5/", allowHosts: []string{"192.168.0.0/16"}, wantErr: false},
		{name: "metadata blocked", url: "http://metadata.google.internal/", wantErr: true},
		{name: "empty", url: "", wantErr: true},
		{name: "ftp", url: "ftp://example.com/", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := AssertNavigateURL(tt.url, tt.allowPrivate, tt.allowHosts)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestSessionStaleRef(t *testing.T) {
	t.Parallel()
	s, err := NewSession(Config{Endpoint: "ws://127.0.0.1:9"})
	require.NoError(t, err)
	s.page = &fakePage{refs: map[string]struct{}{"e0": {}}}
	s.refs = map[string]struct{}{"e0": {}}

	err = s.Click(context.Background(), "e0")
	require.NoError(t, err)

	err = s.Click(context.Background(), "missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stale")

	_, err = s.Navigate(context.Background(), "https://example.com")
	require.NoError(t, err)
	err = s.Click(context.Background(), "e0")
	require.Error(t, err)
}

func TestSessionClosed(t *testing.T) {
	t.Parallel()
	s, err := NewSession(Config{})
	require.NoError(t, err)
	require.NoError(t, s.Close())
	_, err = s.Snapshot(context.Background())
	require.Error(t, err)
}

type fakePage struct {
	refs map[string]struct{}
}

func (*fakePage) Navigate(_ context.Context, rawURL string) (string, error) {
	return rawURL, nil
}

func (f *fakePage) Snapshot(_ context.Context) (string, map[string]struct{}, error) {
	return "tree", f.refs, nil
}

func (*fakePage) Click(_ context.Context, _ string) error { return nil }

func (*fakePage) Type(_ context.Context, _, _ string) error { return nil }

func (*fakePage) Scroll(_ context.Context, _, _ int) error { return nil }

func (*fakePage) Wait(_ context.Context, _ int, _ Quirks) error { return nil }

func (*fakePage) Screenshot(_ context.Context) ([]byte, error) { return []byte("png"), nil }

func (*fakePage) Close() error { return nil }
