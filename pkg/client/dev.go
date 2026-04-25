package client

import (
	"context"

	"github.com/flowline-io/flowbot/pkg/types"
)

// DevClient provides access to the dev API.
type DevClient struct {
	c *Client
}

// ExampleData represents example data from the dev endpoint.
type ExampleData struct {
	Title string `json:"title"`
	CPU   string `json:"cpu"`
	Mem   string `json:"mem"`
	Disk  string `json:"disk"`
}

// Example returns example data for testing purposes.
// This endpoint does not require authentication.
func (d *DevClient) Example(ctx context.Context) (*ExampleData, error) {
	var result types.KV
	err := d.c.Get(ctx, "/service/dev/example", &result)
	if err != nil {
		return nil, err
	}

	return &ExampleData{
		Title: stringOr(result, "title", ""),
		CPU:   stringOr(result, "cpu", ""),
		Mem:   stringOr(result, "mem", ""),
		Disk:  stringOr(result, "disk", ""),
	}, nil
}
