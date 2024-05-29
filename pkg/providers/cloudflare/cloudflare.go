package cloudflare

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
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

func NewCloudflare(token string, zoneID string) *Cloudflare {
	v := &Cloudflare{token: token, zoneID: zoneID}

	v.c = resty.New()
	v.c.SetBaseURL("https://api.cloudflare.com/client/v4/")
	v.c.SetTimeout(time.Minute)

	return v
}

func (v *Cloudflare) GetAnalytics(start, end string) (*AnalyticResponse, error) {
	resp, err := v.c.R().
		SetAuthToken(v.token).
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
