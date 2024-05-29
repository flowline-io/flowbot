package slash

import (
	"time"

	"github.com/go-resty/resty/v2"
)

const (
	ID          = "slash"
	EndpointKey = "endpoint"
)

type Slash struct {
	c *resty.Client
}

func NewSlash(endpoint string) *Slash {
	v := &Slash{}
	v.c = resty.New()
	v.c.SetBaseURL(endpoint)
	v.c.SetTimeout(time.Minute)

	return v
}
