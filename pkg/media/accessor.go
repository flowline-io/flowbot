package media

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/flowline-io/flowbot/pkg/types"
)

// Accessor extends Handler with file-id based open and short-lived signed GET URLs for LLM fetch.
type Accessor interface {
	Handler
	// SignGetURL returns a time-limited absolute URL the model provider can fetch.
	SignGetURL(ctx context.Context, fileID string, ttl time.Duration) (string, error)
	// OpenByID opens stored bytes by file id.
	OpenByID(ctx context.Context, fileID string) (*types.FileDef, ReadSeekCloser, error)
}

// AsAccessor returns Accessor when the handler implements it.
func AsAccessor(h Handler) (Accessor, bool) {
	a, ok := h.(Accessor)
	return a, ok
}

// ReadAll opens a file by id and returns its bytes (caller must use Accessor).
func ReadAll(ctx context.Context, a Accessor, fileID string) (*types.FileDef, []byte, error) {
	if a == nil {
		return nil, nil, fmt.Errorf("media: nil accessor")
	}
	fd, rc, err := a.OpenByID(ctx, fileID)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = rc.Close() }()
	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, nil, fmt.Errorf("media: read file %s: %w", fileID, err)
	}
	return fd, data, nil
}
