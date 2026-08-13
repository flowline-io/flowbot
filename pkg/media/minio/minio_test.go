package minio

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appConfig "github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/types"
)

func withMediaMaxSize(t *testing.T, maxSize int64) {
	t.Helper()
	prev := appConfig.App
	t.Cleanup(func() { appConfig.App = prev })
	require.NoError(t, sonic.Unmarshal(fmt.Appendf(nil, `{"media":{"max_size":%d}}`, maxSize), &appConfig.App))
}

func TestInit_Validation(t *testing.T) {
	tests := []struct {
		name    string
		jsconf  string
		wantErr string
	}{
		{
			name:    "invalid json",
			jsconf:  `{`,
			wantErr: "error parsing config",
		},
		{
			name:    "missing access key",
			jsconf:  `{"secret_access_key":"s","bucket":"b","endpoint":"localhost:9000"}`,
			wantErr: "missing Access Key ID",
		},
		{
			name:    "missing secret key",
			jsconf:  `{"access_key_id":"a","bucket":"b","endpoint":"localhost:9000"}`,
			wantErr: "missing Secret Access Key",
		},
		{
			name:    "missing bucket",
			jsconf:  `{"access_key_id":"a","secret_access_key":"s","endpoint":"localhost:9000"}`,
			wantErr: "missing Bucket",
		},
		{
			name: "defaults serve url before dial",
			jsconf: `{
				"access_key_id":"a",
				"secret_access_key":"s",
				"bucket":"b",
				"endpoint":"127.0.0.1:1",
				"disable_ssl":true
			}`,
			wantErr: "error checking if bucket",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &handler{}
			err := h.Init(tt.jsconf)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
			if strings.Contains(tt.name, "defaults") {
				assert.Equal(t, defaultServeURL, h.conf.ServeURL)
			}
		})
	}
}

func TestUpload_Validation(t *testing.T) {
	withMediaMaxSize(t, 32)

	h := &handler{conf: config{BucketName: "bucket", ServeURL: defaultServeURL}}
	tests := []struct {
		name    string
		fdef    *types.FileDef
		wantErr string
	}{
		{
			name:    "empty file",
			fdef:    &types.FileDef{Size: 0},
			wantErr: "empty file",
		},
		{
			name:    "oversize",
			fdef:    &types.FileDef{Size: 64},
			wantErr: "max file upload size",
		},
		{
			name: "sets defaults then fails without client",
			fdef: &types.FileDef{
				ObjHeader: types.ObjHeader{Id: "id-1"},
				Size:      4,
			},
			wantErr: "failed to create file record",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, n, err := h.Upload(tt.fdef, strings.NewReader("data"))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
			assert.Empty(t, url)
			assert.Zero(t, n)
			if tt.name == "sets defaults then fails without client" {
				assert.Equal(t, "application/octet-stream", tt.fdef.MimeType)
				assert.NotEmpty(t, tt.fdef.Location)
				assert.Contains(t, tt.fdef.Location, "id-1")
			}
		})
	}
}

func TestGetIdFromUrl(t *testing.T) {
	h := &handler{conf: config{ServeURL: "/v0/file/s/"}}
	tests := []struct {
		name string
		url  string
		want types.Uid
	}{
		{name: "extracts id", url: "/v0/file/s/abc123", want: types.Uid("abc123")},
		{name: "wrong prefix", url: "/other/abc123", want: types.ZeroUid},
		{name: "empty", url: "", want: types.ZeroUid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, h.GetIdFromUrl(tt.url))
		})
	}
}
