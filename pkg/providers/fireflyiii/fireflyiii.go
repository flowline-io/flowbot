package fireflyiii

import (
	"fmt"

	"github.com/flowline-io/flowbot/pkg/utils"

	"github.com/flowline-io/flowbot/pkg/providers"
	"resty.dev/v3"
)

const (
	ID          = "fireflyiii"
	EndpointKey = "endpoint"
	TokenKey    = "token"
)

type FireflyIII struct {
	c *resty.Client
}

func GetClient() *FireflyIII {
	endpoint, _ := providers.GetConfig(ID, EndpointKey)
	tokenKey, _ := providers.GetConfig(ID, TokenKey)

	return NewFireflyIII(endpoint.String(), tokenKey.String())
}

func NewFireflyIII(endpoint string, token string) *FireflyIII {
	v := &FireflyIII{}

	v.c = utils.DefaultRestyClient()
	v.c.SetBaseURL(endpoint)
	v.c.SetAuthToken(token)

	return v
}

func (i *FireflyIII) About() (*About, error) {
	resp, err := i.c.R().
		SetResult(&Response{}).
		Get("/v1/about")
	if err != nil {
		return nil, fmt.Errorf("failed to get about: %w", err)
	}

	result := resp.Result().(*Response)
	return ConvertResponseData[About](result, resp.StatusCode())
}

// CurrentUser Returns the currently authenticated user.
func (i *FireflyIII) CurrentUser() (*About, error) {
	resp, err := i.c.R().
		SetResult(&Response{}).
		Get("/v1/about/user")
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	result := resp.Result().(*Response)
	return ConvertResponseData[About](result, resp.StatusCode())
}

// CreateTransaction Creates a new transaction.
func (i *FireflyIII) CreateTransaction(transaction Transaction) (*TransactionResult, error) {
	resp, err := i.c.R().
		SetResult(&Response{}).
		SetBody(transaction).
		Post("/v1/transactions")
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	result := resp.Result().(*Response)
	return ConvertResponseData[TransactionResult](result, resp.StatusCode())
}
