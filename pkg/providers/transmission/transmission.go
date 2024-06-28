package transmission

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/hekmon/transmissionrpc/v3"
)

const (
	ID          = "transmission"
	EndpointKey = "endpoint"
)

type Transmission struct {
	c *transmissionrpc.Client
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
func (v *Transmission) TorrentAddUrl(ctx context.Context, url string) (transmissionrpc.Torrent, error) {
	if !strings.HasPrefix(url, "magnet") {
		return transmissionrpc.Torrent{}, errors.New("only magnet links are supported")
	}

	return v.c.TorrentAdd(ctx, transmissionrpc.TorrentAddPayload{
		Filename: &url,
	})
}
