package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientVersion(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantVer    string
		wantStamp  string
		wantErrSub string
	}{
		{
			name: "success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/version", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"version":"v1.2.3","buildstamp":"stamp"}}`))
			},
			wantVer:   "v1.2.3",
			wantStamp: "stamp",
		},
		{
			name: "failed status",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"failed","message":"boom"}`))
			},
			wantErrSub: "boom",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(tt.handler)
			t.Cleanup(srv.Close)

			c := NewClient(srv.URL, "")
			info, err := c.Version(context.Background())
			if tt.wantErrSub != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrSub)
				assert.Nil(t, info)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, info)
			assert.Equal(t, tt.wantVer, info.Version)
			assert.Equal(t, tt.wantStamp, info.Buildstamp)
		})
	}
}
