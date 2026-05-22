// Package drone implements the Drone CI provider.
package drone

import (
	"fmt"

	"resty.dev/v3"

	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/utils"
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

func NewDrone(endpoint, token string) *Drone {
	v := &Drone{}

	v.c = utils.DefaultRestyClient()
	v.c.SetBaseURL(endpoint)
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

	result, ok := resp.Result().(*Build)
	if !ok {
		return nil, fmt.Errorf("unexpected response type from drone")
	}
	return result, nil
}
