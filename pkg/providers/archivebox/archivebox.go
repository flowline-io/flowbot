package archivebox

import (
	"fmt"

	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/utils"
	"resty.dev/v3"
)

const (
	ID          = "archivebox"
	EndpointKey = "endpoint"
	TokenKey    = "token"
)

type ArchiveBox struct {
	c *resty.Client
}

func GetClient() *ArchiveBox {
	endpoint, _ := providers.GetConfig(ID, EndpointKey)
	tokenKey, _ := providers.GetConfig(ID, TokenKey)
	return NewArchiveBox(endpoint.String(), tokenKey.String())
}

func NewArchiveBox(endpoint string, token string) *ArchiveBox {
	v := &ArchiveBox{}
	v.c = utils.DefaultRestyClient()
	v.c.SetBaseURL(endpoint)
	v.c.SetAuthToken(token)

	return v
}

func (i *ArchiveBox) Add(data Data) (*Response, error) {
	resp, err := i.c.R().
		SetResult(&Response{}).
		SetBody(data).
		Post("/api/v1/cli/add")
	if err != nil {
		return nil, fmt.Errorf("failed to add: %w", err)
	}

	result := resp.Result().(*Response)
	return result, nil
}
