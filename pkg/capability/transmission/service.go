package transmission

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/capability"
)

// AddTorrentInput holds parameters for adding a torrent by URL or magnet link.
type AddTorrentInput struct {
	URL string
}

// StopTorrentsInput holds parameters for stopping torrents by ID.
type StopTorrentsInput struct {
	IDs []int64
}

// RemoveTorrentsInput holds parameters for removing torrents by ID.
type RemoveTorrentsInput struct {
	IDs []int64
}

// Service defines the transmission download capability contract.
type Service interface {
	AddTorrent(ctx context.Context, in AddTorrentInput) (*capability.Torrent, error)
	ListTorrents(ctx context.Context) ([]*capability.Torrent, error)
	StopTorrents(ctx context.Context, in StopTorrentsInput) error
	RemoveTorrents(ctx context.Context, in RemoveTorrentsInput) error
	HealthCheck(ctx context.Context) (bool, error)
}
