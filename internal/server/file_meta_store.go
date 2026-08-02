package server

import (
	"context"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/media"
	"github.com/flowline-io/flowbot/pkg/types"
)

// fileMetaStore adapts store.FileStore to media.FileMetaStore.
type fileMetaStore struct{}

func (fileMetaStore) FileStartUpload(ctx context.Context, fd *types.FileDef) error {
	return store.FileStoreFromDB().FileStartUpload(ctx, fd)
}

func (fileMetaStore) FileFinishUpload(ctx context.Context, fd *types.FileDef, success bool, size int64) (*types.FileDef, error) {
	return store.FileStoreFromDB().FileFinishUpload(ctx, fd, success, size)
}

func (fileMetaStore) FileGet(ctx context.Context, fid string) (*types.FileDef, error) {
	return store.FileStoreFromDB().FileGet(ctx, fid)
}

// WireFileMetaStore injects the store-backed file metadata adapter into media.
func WireFileMetaStore() {
	media.SetFileMetaStore(fileMetaStore{})
}
