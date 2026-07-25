// Package transmission implements the Transmission BitTorrent provider.
package transmission

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/hekmon/transmissionrpc/v3"

	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/utils"
)

const (
	ID          = "transmission"
	EndpointKey = "endpoint"
)

type Transmission struct {
	c *transmissionrpc.Client
}

// GetClient returns a Transmission client from YAML config.
// Returns nil, nil when the endpoint is not configured.
func GetClient() (*Transmission, error) {
	endpoint, _ := providers.GetConfig(ID, EndpointKey)
	if endpoint.String() == "" {
		return nil, nil
	}
	return NewTransmission(endpoint.String())
}

// NewTransmission creates a Transmission client for the given RPC endpoint.
func NewTransmission(endpoint string) (*Transmission, error) {
	if endpoint == "" {
		return nil, nil
	}
	v := &Transmission{}

	e, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	tbt, err := transmissionrpc.New(e, nil)
	if err != nil {
		return nil, err
	}
	v.c = tbt

	return v, nil
}

// TorrentAddFile adds a new torrent by uploading a .torrent file.
//
// ctx is the context for the request.
// filepath is the path to the .torrent file.
// Returns transmissionrpc.Torrent and error.
func (v *Transmission) TorrentAddFile(ctx context.Context, filepath string) (transmissionrpc.Torrent, error) {
	return v.c.TorrentAddFile(ctx, filepath)
}

// TorrentAddUrl adds a torrent to the Transmission client using a magnet link.
//
// ctx - the context for the function.
// url - the magnet link to add.
// (transmissionrpc.Torrent, error) - returns the added torrent or an error.
func (v *Transmission) TorrentAddUrl(ctx context.Context, magnetUrl string) (transmissionrpc.Torrent, error) {
	if strings.HasPrefix(magnetUrl, "magnet") {
		return v.c.TorrentAdd(ctx, transmissionrpc.TorrentAddPayload{
			Filename: &magnetUrl,
		})
	}

	if !isValidRedirect(magnetUrl) {
		return transmissionrpc.Torrent{}, fmt.Errorf("transmission: invalid torrent url")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, magnetUrl, http.NoBody)
	if err != nil {
		return transmissionrpc.Torrent{}, err
	}
	httpClient := &http.Client{
		Transport: utils.HTTPTransport(),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("transmission: too many redirects")
			}
			if !isValidRedirect(req.URL.String()) {
				return fmt.Errorf("transmission: invalid torrent redirect url")
			}
			return nil
		},
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return transmissionrpc.Torrent{}, err
	}
	defer resp.Body.Close()

	tempFile, err := os.CreateTemp("", "torrent-*.torrent")
	if err != nil {
		return transmissionrpc.Torrent{}, err
	}
	defer tempFile.Close()

	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		return transmissionrpc.Torrent{}, err
	}

	return v.c.TorrentAddFile(ctx, tempFile.Name())
}

// isValidRedirect is named for CodeQL's redirect-check barrier (go/request-forgery).
func isValidRedirect(rawURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Host == "" {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

// TorrentGetAll returns all the known fields for all the torrents.
func (v *Transmission) TorrentGetAll(ctx context.Context) ([]transmissionrpc.Torrent, error) {
	return v.c.TorrentGetAll(ctx)
}

// TorrentStopIDs stops torrent(s) which id is in the provided slice.
// Can be one, can be several, can be all (if slice is empty or nil).
func (v *Transmission) TorrentStopIDs(ctx context.Context, ids []int64) error {
	return v.c.TorrentStopIDs(ctx, ids)
}

// TorrentRemove allows to delete one or more torrents only.
func (v *Transmission) TorrentRemove(ctx context.Context, ids []int64) error {
	return v.c.TorrentRemove(ctx, transmissionrpc.TorrentRemovePayload{IDs: ids})
}
