package transmission

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/hekmon/transmissionrpc/v3"
)

const (
	ID          = "transmission"
	EndpointKey = "endpoint"
)

type Transmission struct {
	c *transmissionrpc.Client
}

func GetClient() (*Transmission, error) {
	endpoint, _ := providers.GetConfig(ID, EndpointKey)

	return NewTransmission(endpoint.String())
}

func NewTransmission(endpoint string) (*Transmission, error) {
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

	// download the torrent file from url
	httpClient := &http.Client{}
	resp, err := httpClient.Get(magnetUrl)
	if err != nil {
		return transmissionrpc.Torrent{}, err
	}
	defer resp.Body.Close()

	// store the torrent file in a temporary file
	tempFile, err := os.CreateTemp("", "torrent-*.torrent")
	if err != nil {
		return transmissionrpc.Torrent{}, err
	}
	defer tempFile.Close()

	// copy the contents of the response body to the temporary file
	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		return transmissionrpc.Torrent{}, err
	}

	return v.c.TorrentAddFile(ctx, tempFile.Name())
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
