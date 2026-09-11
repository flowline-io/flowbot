package utils_test

import (
	"net/url"
	"testing"

	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssertPublicHTTPURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		raw          string
		allowPrivate bool
		wantErr      bool
	}{
		{name: "public https", raw: "https://example.com/a", wantErr: false},
		{name: "loopback blocked", raw: "http://127.0.0.1/", wantErr: true},
		{name: "private blocked", raw: "http://10.1.2.3/", wantErr: true},
		{name: "metadata host blocked", raw: "http://metadata.google.internal/", wantErr: true},
		{name: "file scheme", raw: "file:///etc/passwd", wantErr: true},
		{name: "allow private skips checks", raw: "http://10.1.2.3/", allowPrivate: true, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			u, err := url.Parse(tt.raw)
			require.NoError(t, err)
			err = utils.AssertPublicHTTPURL(u, tt.allowPrivate)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
