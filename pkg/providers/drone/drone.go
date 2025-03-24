package drone

import (
	"fmt"
	"time"

	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/go-resty/resty/v2"
)

const (
	ID          = "drone"
	EndpointKey = "endpoint"
	TokenKey    = "token"
)

type Drone struct {
	c *resty.Client
}

func GetClient() *Drone {
	endpoint, _ := providers.GetConfig(ID, EndpointKey)
	tokenKey, _ := providers.GetConfig(ID, TokenKey)

	return NewDrone(endpoint.String(), tokenKey.String())
}

func NewDrone(endpoint string, token string) *Drone {
	v := &Drone{}

	v.c = resty.New()
	v.c.SetBaseURL(endpoint)
	v.c.SetTimeout(time.Minute)
	v.c.SetAuthToken(token)

	return v
}

func (i *Drone) CreateBuild(namespace, name string) (*Build, error) {
	resp, err := i.c.R().
		SetResult(&Build{}).
		SetPathParams(map[string]string{"namespace": namespace, "name": name}).
		Post("/api/repos/{namespace}/{name}/builds")
	if err != nil {
		return nil, fmt.Errorf("failed to create build: %w", err)
	}

	result := resp.Result().(*Build)
	return result, nil
}
