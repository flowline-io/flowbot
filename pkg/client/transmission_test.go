package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransmissionAddTorrent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		req        *AddTorrentRequest
		handler    http.HandlerFunc
		wantID     int64
		wantErr    bool
		errContain string
	}{
		{
			name: "add success",
			req:  &AddTorrentRequest{URL: "magnet:?xt=urn:btih:abc"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/service/transmission/torrents", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"ok","data":{"data":{"id":7,"name":"ubuntu.iso"}}}`))
			},
			wantID: 7,
		},
		{
			name:       "nil request",
			req:        nil,
			wantErr:    true,
			errContain: "request is required",
		},
		{
			name:       "missing url",
			req:        &AddTorrentRequest{},
			wantErr:    true,
			errContain: "url is required",
		},
		{
			name: "api error",
			req:  &AddTorrentRequest{URL: "magnet:?xt=urn:btih:abc"},
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"status":"failed","message":"server error"}`))
			},
			wantErr:    true,
			errContain: "server error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.handler != nil {
					tt.handler(w, r)
				}
			}))
			defer server.Close()

			c := NewClient(server.URL, "token")
			got, err := c.Transmission.AddTorrent(context.Background(), tt.req)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantID, got.ID)
		})
	}
}

func TestTransmissionListStopRemoveHealth(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "list success",
			run: func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/service/transmission/torrents", r.URL.Path)
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"status":"ok","data":{"data":[{"id":1,"name":"a","status":"downloading"}]}}`))
				}))
				defer server.Close()
				c := NewClient(server.URL, "token")
				items, err := c.Transmission.ListTorrents(context.Background())
				require.NoError(t, err)
				require.Len(t, items, 1)
				assert.Equal(t, int64(1), items[0].ID)
			},
		},
		{
			name: "stop success",
			run: func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, http.MethodPost, r.Method)
					assert.Equal(t, "/service/transmission/torrents/stop", r.URL.Path)
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"status":"ok","data":{"data":{"stopped":2}}}`))
				}))
				defer server.Close()
				c := NewClient(server.URL, "token")
				require.NoError(t, c.Transmission.StopTorrents(context.Background(), []int64{1, 2}))
			},
		},
		{
			name: "stop missing ids",
			run: func(t *testing.T) {
				c := NewClient("http://example.invalid", "token")
				err := c.Transmission.StopTorrents(context.Background(), nil)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "ids is required")
			},
		},
		{
			name: "remove success",
			run: func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/service/transmission/torrents/remove", r.URL.Path)
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"status":"ok","data":{"data":{"removed":1}}}`))
				}))
				defer server.Close()
				c := NewClient(server.URL, "token")
				require.NoError(t, c.Transmission.RemoveTorrents(context.Background(), []int64{9}))
			},
		},
		{
			name: "health healthy",
			run: func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/service/transmission/health", r.URL.Path)
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"status":"ok","data":{"data":true}}`))
				}))
				defer server.Close()
				c := NewClient(server.URL, "token")
				ok, err := c.Transmission.Health(context.Background())
				require.NoError(t, err)
				assert.True(t, ok)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t)
		})
	}
}
