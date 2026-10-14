// Package fireflyiii implements the Firefly III finance provider.
package fireflyiii

import (
	"errors"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/utils"

	"resty.dev/v3"

	"github.com/flowline-io/flowbot/pkg/providers"
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
	if endpoint.String() == "" {
		return nil
	}
	return NewFireflyIII(endpoint.String(), tokenKey.String())
}

// NewFireflyIII creates a Firefly III client with the given endpoint and token.
// If endpoint is empty, it returns nil.
func NewFireflyIII(endpoint, token string) *FireflyIII {
	if endpoint == "" {
		return nil
	}
	v := &FireflyIII{}

	v.c = utils.DefaultRestyClient()
	v.c.SetBaseURL(endpoint)
	v.c.SetAuthToken(token)
	// Firefly redirects unauthenticated browser-style requests (no Accept) to /login HTML.
	// Prefer JSON so auth failures surface as 401 with message/exception instead of empty 200 bodies.
	v.c.SetHeader("Accept", "application/json, application/vnd.api+json")

	return v
}

func (i *FireflyIII) About() (*About, error) {
	resp, err := i.c.R().Get("/v1/about")
	if err != nil {
		return nil, fmt.Errorf("failed to get about: %w", err)
	}
	payload, err := decodeResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to get about: %w", err)
	}
	return ConvertResponseData[About](payload, resp.StatusCode())
}

// CurrentUser returns the currently authenticated user.
func (i *FireflyIII) CurrentUser() (*User, error) {
	resp, err := i.c.R().Get("/v1/about/user")
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}
	payload, err := decodeResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}
	return ConvertResponseData[User](payload, resp.StatusCode())
}

// CreateTransaction Creates a new transaction.
func (i *FireflyIII) CreateTransaction(transaction Transaction) (*TransactionResult, error) {
	resp, err := i.c.R().
		SetBody(transaction).
		Post("/v1/transactions")
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}
	payload, err := decodeResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}
	return ConvertResponseData[TransactionResult](payload, resp.StatusCode())
}

func decodeResponse(resp *resty.Response) (*Response, error) {
	if resp == nil {
		return nil, errors.New("response is nil")
	}
	body := resp.Bytes()
	if len(body) == 0 {
		return &Response{}, nil
	}
	var payload Response
	if err := sonic.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &payload, nil
}
