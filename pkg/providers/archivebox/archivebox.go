package archivebox

import (
	"github.com/go-resty/resty/v2"
	"time"
)

const (
	ID          = "archivebox"
	EndpointKey = "endpoint"
)

type ArchiveBox struct {
	c *resty.Client
}

func NewArchiveBox(endpoint string) *ArchiveBox {
	v := &ArchiveBox{}
	v.c = resty.New()
	v.c.SetBaseURL(endpoint)
	v.c.SetTimeout(time.Minute)

	return v
}
