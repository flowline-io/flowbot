package adguard

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"time"
)

const (
	ID          = "adguard_home"
	EndpointKey = "endpoint"
	UsernameKey = "username"
	PasswordKey = "password"
)

type AdGuardHome struct {
	c *resty.Client
}

func NewAdGuardHome(endpoint string, username string, password string) *AdGuardHome {
	v := &AdGuardHome{}

	v.c = resty.New()
	v.c.SetBaseURL(endpoint)
	v.c.SetTimeout(time.Minute)
	v.c.SetBasicAuth(username, password)

	return v
}

func (v *AdGuardHome) GetStatus() (*ServerStatus, error) {
	resp, err := v.c.R().
		SetResult(&ServerStatus{}).
		Get("/status")
	if err != nil {
		return nil, fmt.Errorf("failed to Get DNS server current status and general settings: %w", err)
	}

	result := resp.Result().(*ServerStatus)
	return result, nil
}

func (v *AdGuardHome) GetStats() (*Stats, error) {
	resp, err := v.c.R().
		SetResult(&Stats{}).
		Get("/stats")
	if err != nil {
		return nil, fmt.Errorf("failed to Get DNS server statistics: %w", err)
	}

	result := resp.Result().(*Stats)
	return result, nil
}
