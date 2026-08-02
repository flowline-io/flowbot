package media

import (
	"context"
	"fmt"
	"sync"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

// FileMetaStore persists file metadata used by media handlers.
type FileMetaStore interface {
	FileStartUpload(ctx context.Context, fd *types.FileDef) error
	FileFinishUpload(ctx context.Context, fd *types.FileDef, success bool, size int64) (*types.FileDef, error)
	FileGet(ctx context.Context, fid string) (*types.FileDef, error)
}

var (
	fileMetaStoreMu sync.RWMutex
	fileMetaStore   FileMetaStore

	fileHandlersMu sync.RWMutex
	fileHandlers   = map[string]Handler{}

	// FileSystem is the active media handler after UseHandler succeeds.
	FileSystem Handler
)

// SetFileMetaStore wires the persistence backend used by media handlers.
func SetFileMetaStore(s FileMetaStore) {
	fileMetaStoreMu.Lock()
	defer fileMetaStoreMu.Unlock()
	fileMetaStore = s
}

// GetFileMetaStore returns the injected file metadata store.
func GetFileMetaStore() FileMetaStore {
	fileMetaStoreMu.RLock()
	defer fileMetaStoreMu.RUnlock()
	return fileMetaStore
}

func requireFileMetaStore() (FileMetaStore, error) {
	s := GetFileMetaStore()
	if s == nil {
		return nil, fmt.Errorf("media: file meta store is not configured")
	}
	return s, nil
}

// RegisterHandler saves a reference to a media upload/download handler.
func RegisterHandler(name string, mh Handler) {
	fileHandlersMu.Lock()
	defer fileHandlersMu.Unlock()
	if mh == nil {
		flog.Fatal("media.RegisterHandler: handler is nil")
	}
	if _, dup := fileHandlers[name]; dup {
		flog.Fatal("media.RegisterHandler: called twice for handler %s", name)
	}
	fileHandlers[name] = mh
	flog.Info("media: handler '%s' registered", name)
}

// UseHandler sets the named media handler as default and initializes it.
func UseHandler(name, mediaConfig string) error {
	fileHandlersMu.RLock()
	mediaHandler := fileHandlers[name]
	fileHandlersMu.RUnlock()
	if mediaHandler == nil {
		return fmt.Errorf("unknown handler %s", name)
	}
	FileSystem = mediaHandler
	return mediaHandler.Init(mediaConfig)
}

// StartUpload records the start of a file upload via the injected store.
func StartUpload(ctx context.Context, fd *types.FileDef) error {
	s, err := requireFileMetaStore()
	if err != nil {
		return err
	}
	return s.FileStartUpload(ctx, fd)
}

// FinishUpload finalizes a file upload via the injected store.
func FinishUpload(ctx context.Context, fd *types.FileDef, success bool, size int64) (*types.FileDef, error) {
	s, err := requireFileMetaStore()
	if err != nil {
		return nil, err
	}
	return s.FileFinishUpload(ctx, fd, success, size)
}

// GetFile loads file metadata via the injected store.
func GetFile(ctx context.Context, fid string) (*types.FileDef, error) {
	s, err := requireFileMetaStore()
	if err != nil {
		return nil, err
	}
	return s.FileGet(ctx, fid)
}
