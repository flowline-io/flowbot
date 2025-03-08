package archivebox

import (
	"time"

	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/go-resty/resty/v2"
)

const (
	ID          = "archivebox"
	EndpointKey = "endpoint"
)

type ArchiveBox struct {
	c *resty.Client
}

func GetClient() *ArchiveBox {
	endpoint, _ := providers.GetConfig(ID, EndpointKey)

	return NewArchiveBox(endpoint.String())
}

func NewArchiveBox(endpoint string) *ArchiveBox {
	v := &ArchiveBox{}
	v.c = resty.New()
	v.c.SetBaseURL(endpoint)
	v.c.SetTimeout(time.Minute)

	return v
}
