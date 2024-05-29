package adguard

import (
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

func (v *AdGuardHome) GetStatus() (*StatusResponse, error) {
	resp, err := v.c.R().
		SetResult(&StatusResponse{}).
		Get("/status")
	if err != nil {
		return nil, err
	}
	return resp.Result().(*StatusResponse), nil
}

func (v *AdGuardHome) GetStats() (*StatisticsResponse, error) {
	resp, err := v.c.R().
		SetResult(&StatisticsResponse{}).
		Get("/stats")
	if err != nil {
		return nil, err
	}
	return resp.Result().(*StatisticsResponse), nil
}
