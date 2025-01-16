package kanboard

import (
	"time"

	"github.com/go-resty/resty/v2"
)

const (
	ID              = "kanboard"
	EndpointKey     = "endpoint"
	ApikeyKey       = "api_key"
	WebhookTokenKey = "webhook_token"
)

type Kanboard struct {
	key string
	c   *resty.Client
}

func NewKanboard(endpoint string, key string) *Kanboard {
	v := &Kanboard{key: key}
	v.c = resty.New()
	v.c.SetBaseURL(endpoint)
	v.c.SetTimeout(time.Minute)

	return v
}
