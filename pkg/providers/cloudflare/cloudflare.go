package cloudflare

import (
	"fmt"
	"net/http"

	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/utils"
	"resty.dev/v3"
)

const (
	ID        = "cloudflare"
	Token     = "token"
	ZoneID    = "zone_id"
	AccountID = "account_id"
)

type Cloudflare struct {
	c      *resty.Client
	token  string
	zoneID string
}

func GetClient() *Cloudflare {
	token, _ := providers.GetConfig(ID, Token)
	zoneID, _ := providers.GetConfig(ID, ZoneID)

	return NewCloudflare(token.String(), zoneID.String())
}

func NewCloudflare(token string, zoneID string) *Cloudflare {
	v := &Cloudflare{token: token, zoneID: zoneID}

	v.c = utils.DefaultRestyClient()
	v.c.SetBaseURL("https://api.cloudflare.com/client/v4/")
	v.c.SetAuthToken(v.token)

	return v
}

func (v *Cloudflare) GetAnalytics(start, end string) (*AnalyticResponse, error) {
	resp, err := v.c.R().
		SetResult(&AnalyticResponse{}).
		SetBody(map[string]interface{}{
			"query": fmt.Sprintf(`
query
{
  viewer
  {
    zones(filter: { zoneTag: "%s"})
    {
      firewallEventsAdaptive(
          filter: {
            datetime_gt: "%s",
            datetime_lt: "%s"
          },
          limit: 2,
          orderBy: [datetime_DESC, rayName_DESC])
      {
        action
        datetime
        rayName
        clientRequestHTTPHost
        userAgent
      }
    }
  }
}
`, v.zoneID, start, end),
		}).
		Post("/graphql")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		result := resp.Result().(*AnalyticResponse)
		return result, nil
	}
	return nil, fmt.Errorf("cloudflare api error %d", resp.StatusCode())
}
