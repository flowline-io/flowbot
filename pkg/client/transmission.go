package client

import (
	"context"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/capability"
)

// TransmissionClient provides access to the Transmission download API.
type TransmissionClient struct {
	c *Client
}

// AddTorrentRequest is the request body for adding a torrent.
type AddTorrentRequest struct {
	URL string `json:"url"`
}

// TorrentsActionRequest is the request body for stop/remove operations.
type TorrentsActionRequest struct {
	IDs []int64 `json:"ids"`
}

// TorrentItemResult holds a single torrent extracted from InvokeResult.
type TorrentItemResult struct {
	Item capability.Torrent `json:"data"`
}

// TorrentListResult holds torrents extracted from InvokeResult.
type TorrentListResult struct {
	Items []*capability.Torrent `json:"data"`
}

// TorrentsActionResult holds stop/remove counts extracted from InvokeResult.
type TorrentsActionResult struct {
	Data map[string]any `json:"data"`
}

// TransmissionHealthResult holds the health check result extracted from InvokeResult.
type TransmissionHealthResult struct {
	Healthy bool `json:"data"`
}

// AddTorrent adds a torrent by magnet link or HTTP(S) .torrent URL.
func (t *TransmissionClient) AddTorrent(ctx context.Context, req *AddTorrentRequest) (*capability.Torrent, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	if req.URL == "" {
		return nil, fmt.Errorf("url is required")
	}
	var result TorrentItemResult
	err := t.c.Post(ctx, "/service/transmission/torrents", req, &result)
	if err != nil {
		return nil, err
	}
	return &result.Item, nil
}

// ListTorrents returns all torrents.
func (t *TransmissionClient) ListTorrents(ctx context.Context) ([]*capability.Torrent, error) {
	var result TorrentListResult
	err := t.c.Get(ctx, "/service/transmission/torrents", &result)
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

// StopTorrents stops torrents by ID.
func (t *TransmissionClient) StopTorrents(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return fmt.Errorf("ids is required")
	}
	var result TorrentsActionResult
	return t.c.Post(ctx, "/service/transmission/torrents/stop", &TorrentsActionRequest{IDs: ids}, &result)
}

// RemoveTorrents removes torrents by ID.
func (t *TransmissionClient) RemoveTorrents(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return fmt.Errorf("ids is required")
	}
	var result TorrentsActionResult
	return t.c.Post(ctx, "/service/transmission/torrents/remove", &TorrentsActionRequest{IDs: ids}, &result)
}

// Health checks whether the Transmission backend is reachable.
func (t *TransmissionClient) Health(ctx context.Context) (bool, error) {
	var result TransmissionHealthResult
	err := t.c.Get(ctx, "/service/transmission/health", &result)
	if err != nil {
		return false, err
	}
	return result.Healthy, nil
}
