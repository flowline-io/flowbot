// Package transmission implements the Transmission adapter for the download capability.
package transmission

import (
	"context"

	"github.com/hekmon/transmissionrpc/v3"

	"github.com/flowline-io/flowbot/pkg/capability"
	provider "github.com/flowline-io/flowbot/pkg/providers/transmission"
	"github.com/flowline-io/flowbot/pkg/types"
)

// client defines the subset of provider.Transmission methods used by this adapter.
type client interface {
	TorrentAddUrl(ctx context.Context, magnetUrl string) (transmissionrpc.Torrent, error)
	TorrentGetAll(ctx context.Context) ([]transmissionrpc.Torrent, error)
	TorrentStopIDs(ctx context.Context, ids []int64) error
	TorrentRemove(ctx context.Context, ids []int64) error
}

// Adapter implements Service using the Transmission provider client.
type Adapter struct {
	client client
}

// New creates an Adapter using the default provider client (reads config from YAML).
// Returns nil when the provider is not configured.
func New() Service {
	c, err := provider.GetClient()
	if err != nil || c == nil {
		return nil
	}
	return NewWithClient(c)
}

// NewWithClient creates an Adapter with a specific client, useful for testing.
func NewWithClient(c client) Service {
	return &Adapter{client: c}
}

// AddTorrent adds a torrent via magnet link or HTTP(S) .torrent URL.
func (a *Adapter) AddTorrent(ctx context.Context, in AddTorrentInput) (*capability.Torrent, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if in.URL == "" {
		return nil, types.Errorf(types.ErrInvalidArgument, "url is required")
	}
	torrent, err := a.client.TorrentAddUrl(ctx, in.URL)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "transmission add torrent failed", err)
	}
	return toTorrent(torrent), nil
}

// ListTorrents returns all torrents known to Transmission.
func (a *Adapter) ListTorrents(ctx context.Context) ([]*capability.Torrent, error) {
	if err := ctx.Err(); err != nil {
		return nil, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	list, err := a.client.TorrentGetAll(ctx)
	if err != nil {
		return nil, types.WrapError(types.ErrProvider, "transmission list torrents failed", err)
	}
	out := make([]*capability.Torrent, 0, len(list))
	for _, item := range list {
		out = append(out, toTorrent(item))
	}
	return out, nil
}

// StopTorrents stops one or more torrents by ID.
func (a *Adapter) StopTorrents(ctx context.Context, in StopTorrentsInput) error {
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if len(in.IDs) == 0 {
		return types.Errorf(types.ErrInvalidArgument, "ids is required")
	}
	if err := a.client.TorrentStopIDs(ctx, in.IDs); err != nil {
		return types.WrapError(types.ErrProvider, "transmission stop torrents failed", err)
	}
	return nil
}

// RemoveTorrents removes one or more torrents by ID (data on disk is kept).
func (a *Adapter) RemoveTorrents(ctx context.Context, in RemoveTorrentsInput) error {
	if err := ctx.Err(); err != nil {
		return types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	if len(in.IDs) == 0 {
		return types.Errorf(types.ErrInvalidArgument, "ids is required")
	}
	if err := a.client.TorrentRemove(ctx, in.IDs); err != nil {
		return types.WrapError(types.ErrProvider, "transmission remove torrents failed", err)
	}
	return nil
}

// HealthCheck reports whether the Transmission backend is reachable.
func (a *Adapter) HealthCheck(ctx context.Context) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, types.WrapError(types.ErrTimeout, "context canceled", err)
	}
	_, err := a.client.TorrentGetAll(ctx)
	if err != nil {
		return false, types.WrapError(types.ErrProvider, "transmission health check failed", err)
	}
	return true, nil
}

func toTorrent(t transmissionrpc.Torrent) *capability.Torrent {
	out := &capability.Torrent{}
	if t.ID != nil {
		out.ID = *t.ID
	}
	if t.Name != nil {
		out.Name = *t.Name
	}
	if t.Status != nil {
		out.Status = t.Status.String()
	}
	if t.PercentDone != nil {
		out.PercentDone = *t.PercentDone
	}
	if t.RateDownload != nil {
		out.RateDownload = *t.RateDownload
	}
	if t.RateUpload != nil {
		out.RateUpload = *t.RateUpload
	}
	if t.DownloadDir != nil {
		out.DownloadDir = *t.DownloadDir
	}
	if t.HashString != nil {
		out.HashString = *t.HashString
	}
	if t.ErrorString != nil {
		out.ErrorString = *t.ErrorString
	}
	return out
}

// Compile-time interface check.
var _ Service = (*Adapter)(nil)
