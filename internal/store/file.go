package store

import (
	"context"
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/gen/fileupload"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/types"
)

// FileStore persists file upload records.
type FileStore struct {
	client *gen.Client
}

// NewFileStore creates a FileStore with the given ent client.
func NewFileStore(client *gen.Client) *FileStore {
	return &FileStore{client: client}
}

// FileStoreFromDB returns a FileStore using the global database client.
func FileStoreFromDB() *FileStore {
	return NewFileStore(ClientFromDB())
}

// Client returns the underlying ent client.
func (s *FileStore) Client() *gen.Client {
	if s == nil {
		return nil
	}
	return s.client
}

// FileStartUpload start upload a file upload.
func (s *FileStore) FileStartUpload(ctx context.Context, fd *types.FileDef) error {
	_, err := s.client.Fileupload.Create().
		SetUID(fd.User).
		SetFid(fd.Id).
		SetName(fd.Name).
		SetMimetype(fd.MimeType).
		SetSize(fd.Size).
		SetLocation(fd.Location).
		SetState(int(schema.FileStart)).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("postgres: file start upload: %w", err)
	}
	return nil
}

// FileFinishUpload finish upload a file upload.
func (s *FileStore) FileFinishUpload(ctx context.Context, fd *types.FileDef, success bool, size int64) (*types.FileDef, error) {
	st := int(schema.FileFailed)
	if success {
		st = int(schema.FileFinish)
	}
	_, err := s.client.Fileupload.Update().
		Where(fileupload.FidEQ(fd.Id)).
		SetSize(size).
		SetState(st).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: file finish upload: %w", err)
	}
	return s.FileGet(ctx, fd.Id)
}

// FileGet get a file upload.
func (s *FileStore) FileGet(ctx context.Context, fid string) (*types.FileDef, error) {
	u, err := s.client.Fileupload.Query().Where(fileupload.FidEQ(fid)).Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, types.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: file get: %w", err)
	}
	return entFileuploadToFileDef(u), nil
}

// FileDeleteUnused delete unused a file upload.
func (s *FileStore) FileDeleteUnused(ctx context.Context, olderThan time.Time, limit int) ([]string, error) {
	q := s.client.Fileupload.Query().
		Where(fileupload.StateEQ(int(schema.FileFinish)))
	if !olderThan.IsZero() {
		q = q.Where(fileupload.UpdatedAtLT(olderThan))
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	files, err := q.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: file delete unused query: %w", err)
	}
	locations := make([]string, 0, len(files))
	for _, f := range files {
		locations = append(locations, f.Location)
	}
	if len(files) > 0 {
		ids := make([]int64, len(files))
		for i, f := range files {
			ids[i] = f.ID
		}
		_, err := s.client.Fileupload.Delete().Where(fileupload.IDIn(ids...)).Exec(ctx)
		if err != nil {
			return locations, fmt.Errorf("postgres: file delete unused: %w", err)
		}
	}
	return locations, nil
}

func entFileuploadToFileDef(f *gen.Fileupload) *types.FileDef {
	return &types.FileDef{
		ObjHeader: types.ObjHeader{
			Id:        f.Fid,
			CreatedAt: f.CreatedAt,
			UpdatedAt: f.UpdatedAt,
		},
		Name:     f.Name,
		MimeType: f.Mimetype,
		Size:     f.Size,
		Location: f.Location,
		User:     f.UID,
	}
}
