package transmission

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/providers"
)

const testSessionID = "test-session-id"

type rpcRequest struct {
	Method    string         `json:"method"`
	Arguments map[string]any `json:"arguments"`
	Tag       int            `json:"tag"`
}

func writeRPCResult(w http.ResponseWriter, tag int, arguments map[string]any) {
	payload := map[string]any{"result": "success", "tag": tag}
	if len(arguments) > 0 {
		payload["arguments"] = arguments
	}
	w.Header().Set("Content-Type", "application/json")
	_ = sonic.ConfigDefault.NewEncoder(w).Encode(payload)
}

func newTransmissionRPCServer(t *testing.T, handler func(method string, args map[string]any) map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := r.Header.Get("X-Transmission-Session-Id")
		if session == "" {
			w.Header().Set("X-Transmission-Session-Id", testSessionID)
			w.WriteHeader(http.StatusConflict)
			return
		}
		assert.Equal(t, testSessionID, session)

		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		var req rpcRequest
		assert.NoError(t, sonic.Unmarshal(body, &req))

		writeRPCResult(w, req.Tag, handler(req.Method, req.Arguments))
	}))
}

func TestGetClient(t *testing.T) {
	tests := []struct {
		name    string
		configs json.RawMessage
		wantErr bool
	}{
		{name: "empty endpoint still constructs client", configs: json.RawMessage(`{}`), wantErr: false},
		{name: "configured endpoint", configs: json.RawMessage(`{"transmission":{"endpoint":"http://127.0.0.1:9091/transmission/rpc"}}`), wantErr: false},
		{name: "invalid endpoint", configs: json.RawMessage(`{"transmission":{"endpoint":"://bad"}}`), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providers.Configs = tt.configs
			c, err := GetClient()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, c)
		})
	}
}

func TestTransmission_TorrentGetAll(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		torrents []map[string]any
		wantLen  int
		wantErr  bool
	}{
		{name: "returns torrents", torrents: []map[string]any{{"id": 1, "name": "a.torrent"}}, wantLen: 1},
		{name: "empty list", torrents: []map[string]any{}, wantLen: 0},
		{name: "rpc unreachable", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.wantErr {
				tr, err := NewTransmission("http://127.0.0.2:1/transmission/rpc")
				require.NoError(t, err)
				_, err = tr.TorrentGetAll(context.Background())
				assert.Error(t, err)
				return
			}
			srv := newTransmissionRPCServer(t, func(method string, _ map[string]any) map[string]any {
				assert.Equal(t, "torrent-get", method)
				return map[string]any{"torrents": tt.torrents}
			})
			defer srv.Close()

			tr, err := NewTransmission(srv.URL)
			require.NoError(t, err)

			list, err := tr.TorrentGetAll(context.Background())
			require.NoError(t, err)
			assert.Len(t, list, tt.wantLen)
		})
	}
}

func TestTransmission_TorrentAddUrl_Magnet(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{name: "adds magnet link", url: "magnet:?xt=urn:btih:abc"},
		{name: "adds another magnet", url: "magnet:?xt=urn:btih:def"},
		{name: "invalid server", url: "magnet:?xt=urn:btih:bad", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.wantErr {
				tr, err := NewTransmission("http://127.0.0.2:1/transmission/rpc")
				require.NoError(t, err)
				_, err = tr.TorrentAddUrl(context.Background(), tt.url)
				assert.Error(t, err)
				return
			}
			srv := newTransmissionRPCServer(t, func(method string, args map[string]any) map[string]any {
				assert.Equal(t, "torrent-add", method)
				assert.NotNil(t, args["filename"])
				return map[string]any{"torrent-added": map[string]any{"id": 1, "name": "added"}}
			})
			defer srv.Close()

			tr, err := NewTransmission(srv.URL)
			require.NoError(t, err)

			torrent, err := tr.TorrentAddUrl(context.Background(), tt.url)
			require.NoError(t, err)
			require.NotNil(t, torrent.ID)
			assert.Equal(t, int64(1), *torrent.ID)
		})
	}
}

func TestTransmission_TorrentStopIDs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		ids     []int64
		wantErr bool
	}{
		{name: "stops torrent ids", ids: []int64{1, 2}},
		{name: "empty ids", ids: nil},
		{name: "unreachable server", ids: []int64{1}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.wantErr {
				tr, err := NewTransmission("http://127.0.0.2:1/transmission/rpc")
				require.NoError(t, err)
				err = tr.TorrentStopIDs(context.Background(), tt.ids)
				assert.Error(t, err)
				return
			}
			srv := newTransmissionRPCServer(t, func(method string, _ map[string]any) map[string]any {
				assert.Equal(t, "torrent-stop", method)
				return nil
			})
			defer srv.Close()

			tr, err := NewTransmission(srv.URL)
			require.NoError(t, err)
			require.NoError(t, tr.TorrentStopIDs(context.Background(), tt.ids))
		})
	}
}

func TestTransmission_TorrentRemove(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		ids     []int64
		wantErr bool
	}{
		{name: "removes torrent", ids: []int64{9}},
		{name: "removes multiple", ids: []int64{1, 2, 3}},
		{name: "unreachable server", ids: []int64{1}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.wantErr {
				tr, err := NewTransmission("http://127.0.0.2:1/transmission/rpc")
				require.NoError(t, err)
				err = tr.TorrentRemove(context.Background(), tt.ids)
				assert.Error(t, err)
				return
			}
			srv := newTransmissionRPCServer(t, func(method string, _ map[string]any) map[string]any {
				assert.Equal(t, "torrent-remove", method)
				return nil
			})
			defer srv.Close()

			tr, err := NewTransmission(srv.URL)
			require.NoError(t, err)
			require.NoError(t, tr.TorrentRemove(context.Background(), tt.ids))
		})
	}
}

func TestTransmission_TorrentAddUrl_HTTPDownload(t *testing.T) {
	t.Parallel()
	torrentBody := []byte("d8:announce...")
	fileSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/x-bittorrent")
		_, _ = w.Write(torrentBody)
	}))
	defer fileSrv.Close()

	rpcSrv := newTransmissionRPCServer(t, func(method string, _ map[string]any) map[string]any {
		if method == "torrent-add" {
			return map[string]any{"torrent-added": map[string]any{"id": 5, "name": "downloaded.torrent"}}
		}
		return nil
	})
	defer rpcSrv.Close()

	tr, err := NewTransmission(rpcSrv.URL)
	require.NoError(t, err)

	torrent, err := tr.TorrentAddUrl(context.Background(), fileSrv.URL)
	require.NoError(t, err)
	require.NotNil(t, torrent.ID)
	assert.Equal(t, int64(5), *torrent.ID)
}
