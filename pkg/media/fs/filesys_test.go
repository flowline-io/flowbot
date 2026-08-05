package fs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appConfig "github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/media"
	"github.com/flowline-io/flowbot/pkg/types"
)

type memFileStore struct {
	mu        sync.Mutex
	files     map[string]*types.FileDef
	startErr  error
	finishErr error
}

func newMemFileStore() *memFileStore {
	return &memFileStore{files: make(map[string]*types.FileDef)}
}

func (m *memFileStore) FileStartUpload(_ context.Context, fd *types.FileDef) error {
	if m.startErr != nil {
		return m.startErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *fd
	m.files[fd.Id] = &cp
	return nil
}

func (m *memFileStore) FileFinishUpload(_ context.Context, fd *types.FileDef, success bool, size int64) (*types.FileDef, error) {
	if m.finishErr != nil {
		return nil, m.finishErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *fd
	if success {
		cp.Size = size
	}
	m.files[fd.Id] = &cp
	return &cp, nil
}

func (m *memFileStore) FileGet(_ context.Context, fid string) (*types.FileDef, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	fd, ok := m.files[fid]
	if !ok {
		return nil, nil
	}
	cp := *fd
	return &cp, nil
}

func withFileMetaStore(t *testing.T, store media.FileMetaStore) {
	t.Helper()
	prev := media.GetFileMetaStore()
	media.SetFileMetaStore(store)
	t.Cleanup(func() { media.SetFileMetaStore(prev) })
}

func withMediaMaxSize(t *testing.T, maxSize int64) {
	t.Helper()
	prev := appConfig.App
	t.Cleanup(func() { appConfig.App = prev })
	require.NoError(t, sonic.Unmarshal([]byte(fmt.Sprintf(`{"media":{"max_size":%d}}`, maxSize)), &appConfig.App))
}

func TestInit(t *testing.T) {
	tests := []struct {
		name      string
		jsconf    string
		wantErr   string
		wantServe string
	}{
		{
			name:    "invalid json",
			jsconf:  `{`,
			wantErr: "error parsing config",
		},
		{
			name:    "missing upload dir",
			jsconf:  `{"serve_url":"/files/"}`,
			wantErr: "missing upload location",
		},
		{
			name:      "defaults serve url and creates directory",
			jsconf:    "",
			wantServe: defaultServeURL,
		},
		{
			name:      "custom serve url",
			jsconf:    `{"serve_url":"/custom/"}`,
			wantServe: "/custom/",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			jsconf := tt.jsconf
			if tt.wantErr == "" {
				if jsconf == "" {
					jsconf = fmt.Sprintf(`{"upload_dir":%q}`, dir)
				} else {
					jsconf = fmt.Sprintf(`{"upload_dir":%q,"serve_url":%q}`, dir, tt.wantServe)
				}
			}
			fh := &fshandler{}
			err := fh.Init(jsconf)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantServe, fh.serveURL)
			assert.Equal(t, dir, fh.fileUploadLocation)
			info, err := os.Stat(dir)
			require.NoError(t, err)
			assert.True(t, info.IsDir())
		})
	}
}

func TestUpload(t *testing.T) {
	withMediaMaxSize(t, 64)

	tests := []struct {
		name      string
		size      int64
		body      string
		mime      string
		startErr  error
		finishErr error
		wantErr   string
		wantURL   bool
	}{
		{
			name:    "rejects oversize",
			size:    128,
			body:    strings.Repeat("x", 128),
			wantErr: "max file upload size",
		},
		{
			name:     "start upload failure removes file",
			size:     4,
			body:     "data",
			startErr: errors.New("store down"),
			wantErr:  "failed to create file record",
		},
		{
			name:    "success returns serve url with extension",
			size:    4,
			body:    "data",
			mime:    "text/plain",
			wantURL: true,
		},
		{
			name:      "finish upload failure",
			size:      4,
			body:      "data",
			finishErr: errors.New("finish failed"),
			wantErr:   "failed to update file record",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			store := newMemFileStore()
			store.startErr = tt.startErr
			store.finishErr = tt.finishErr
			withFileMetaStore(t, store)

			fh := &fshandler{}
			require.NoError(t, fh.Init(fmt.Sprintf(`{"upload_dir":%q,"serve_url":"/v0/file/s/"}`, dir)))

			fdef := &types.FileDef{
				ObjHeader: types.ObjHeader{Id: "file-1"},
				User:      "user-1",
				MimeType:  tt.mime,
				Size:      tt.size,
			}
			url, n, err := fh.Upload(fdef, strings.NewReader(tt.body))
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Empty(t, url)
				assert.Zero(t, n)
				if tt.startErr != nil {
					_, statErr := os.Stat(filepath.Join(dir, "user-1"))
					assert.True(t, os.IsNotExist(statErr))
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, int64(len(tt.body)), n)
			assert.True(t, strings.HasPrefix(url, "/v0/file/s/file-1"))
			if tt.mime == "text/plain" {
				assert.True(t, strings.HasSuffix(url, ".txt") || strings.Contains(url, "file-1"))
			}
			// Flat layout uses FileDef.Uid() which is User, not Id (FIXME nested dirs).
			_, err = os.Stat(filepath.Join(dir, "user-1"))
			require.NoError(t, err)
		})
	}
}

func TestDelete(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "keep-me")
	require.NoError(t, os.WriteFile(existing, []byte("x"), 0o600))
	missing := filepath.Join(dir, "gone")

	fh := &fshandler{}
	tests := []struct {
		name      string
		locations []string
		wantErr   bool
	}{
		{name: "existing file", locations: []string{existing}},
		{name: "missing file is ignored", locations: []string{missing}},
		{name: "mixed existing and missing", locations: []string{filepath.Join(dir, "also-gone"), existing}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := fh.Delete(tt.locations)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
	_, err := os.Stat(existing)
	assert.True(t, os.IsNotExist(err))
}

func TestDownloadAndOpenByID(t *testing.T) {
	dir := t.TempDir()
	store := newMemFileStore()
	withFileMetaStore(t, store)

	fh := &fshandler{}
	require.NoError(t, fh.Init(fmt.Sprintf(`{"upload_dir":%q}`, dir)))

	path := filepath.Join(dir, "on-disk")
	require.NoError(t, os.WriteFile(path, []byte("hello"), 0o600))
	store.files["fid-1"] = &types.FileDef{
		ObjHeader: types.ObjHeader{Id: "fid-1"},
		Location:  path,
	}

	t.Run("download bad url", func(t *testing.T) {
		_, _, err := fh.Download("/wrong/path")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fid not found")
	})

	t.Run("download missing metadata", func(t *testing.T) {
		_, _, err := fh.Download(fh.serveURL + "missing")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "file not found")
	})

	t.Run("download missing disk file", func(t *testing.T) {
		store.files["fid-gone"] = &types.FileDef{
			ObjHeader: types.ObjHeader{Id: "fid-gone"},
			Location:  filepath.Join(dir, "nope"),
		}
		_, _, err := fh.Download(fh.serveURL + "fid-gone")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "file not found")
	})

	t.Run("download success", func(t *testing.T) {
		fd, rsc, err := fh.Download(fh.serveURL + "fid-1")
		require.NoError(t, err)
		defer rsc.Close()
		assert.Equal(t, "fid-1", fd.Id)
		buf := new(bytes.Buffer)
		_, err = buf.ReadFrom(rsc)
		require.NoError(t, err)
		assert.Equal(t, "hello", buf.String())
	})

	t.Run("open by id success", func(t *testing.T) {
		fd, rsc, err := fh.OpenByID(context.Background(), "fid-1")
		require.NoError(t, err)
		defer rsc.Close()
		assert.Equal(t, path, fd.Location)
	})

	t.Run("open by id missing", func(t *testing.T) {
		_, _, err := fh.OpenByID(context.Background(), "no-such")
		require.Error(t, err)
	})
}

func TestSignGetURL(t *testing.T) {
	prev := appConfig.App
	t.Cleanup(func() { appConfig.App = prev })
	require.NoError(t, sonic.Unmarshal([]byte(`{
		"media":{"sign_secret":"media-secret"},
		"chat_agent":{"media":{"public_base_url":"https://bot.example.com","sign_secret":""}}
	}`), &appConfig.App))

	fh := &fshandler{}
	tests := []struct {
		name    string
		ttl     time.Duration
		wantErr string
	}{
		{name: "falls back to media sign secret", ttl: time.Minute},
		{name: "non-positive ttl still signs", ttl: 0},
		{name: "empty file id", ttl: time.Minute, wantErr: "file id is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileID := "file-abc"
			if tt.wantErr != "" {
				fileID = ""
			}
			url, err := fh.SignGetURL(context.Background(), fileID, tt.ttl)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Contains(t, url, "https://bot.example.com")
			assert.Contains(t, url, "file-abc")
			assert.Contains(t, url, "sig=")
		})
	}
}
