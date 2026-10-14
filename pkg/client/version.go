package client

import "context"

// VersionInfo is the build identity returned by GET /version.
type VersionInfo struct {
	Version    string `json:"version"`
	Buildstamp string `json:"buildstamp"`
}

// Version returns the remote Flowbot server build version.
// The endpoint is public; an access token is not required.
func (c *Client) Version(ctx context.Context) (*VersionInfo, error) {
	var result VersionInfo
	if err := c.Get(ctx, "/version", &result); err != nil {
		return nil, err
	}
	return &result, nil
}
