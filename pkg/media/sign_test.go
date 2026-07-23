package media_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/flowline-io/flowbot/pkg/media"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildAndVerifySignedURL(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 24, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		base    string
		secret  string
		fileID  string
		wantErr string
	}{
		{name: "ok", base: "https://example.com", secret: "sec", fileID: "abc123"},
		{name: "missing base", base: "", secret: "sec", fileID: "abc", wantErr: "public_base_url"},
		{name: "missing secret", base: "https://example.com", secret: "", fileID: "abc", wantErr: "sign secret"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := media.BuildSignedURL(tt.base, tt.secret, tt.fileID, time.Hour, now)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Contains(t, got, media.SignedPathPrefix+tt.fileID)
			assert.Contains(t, got, "sig=")
			exp := now.Add(time.Hour).Unix()
			sig := media.SignFile(tt.secret, tt.fileID, exp)
			require.NoError(t, media.VerifySignedRequest(tt.secret, tt.fileID, strconv.FormatInt(exp, 10), sig, now))
			require.Error(t, media.VerifySignedRequest(tt.secret, tt.fileID, "1", sig, now))
		})
	}
}
