package slack

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/gofiber/fiber/v3"
	"resty.dev/v3"
)

const (
	ID              = "slack"
	ClientIdKey     = "id"
	ClientSecretKey = "secret"
)

// Slack implements the OAuthProvider interface for Slack's OAuth v2 flow
// using the Sign in with Slack (identity.*) scopes.
type Slack struct {
	c            *resty.Client
	clientId     string
	clientSecret string
	redirectURI  string
	accessToken  string
	state        string
}

// NewSlack creates a new Slack OAuth provider instance.
func NewSlack(clientId, clientSecret, redirectURI, accessToken string) *Slack {
	v := &Slack{
		clientId:     clientId,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
		accessToken:  accessToken,
	}

	v.c = utils.DefaultRestyClient()
	v.c.SetBaseURL("https://slack.com/api")

	return v
}

// SetState sets the OAuth state parameter for CSRF protection.
func (v *Slack) SetState(state string) {
	v.state = state
}

// GetAuthorizeURL returns the Slack OAuth v2 authorization URL with
// identity.basic and identity.avatar user scopes.
func (v *Slack) GetAuthorizeURL() string {
	params := url.Values{}
	params.Set("client_id", v.clientId)
	params.Set("user_scope", "identity.basic,identity.avatar")
	params.Set("redirect_uri", v.redirectURI)
	if v.state != "" {
		params.Set("state", v.state)
	}
	return "https://slack.com/oauth/v2/authorize?" + params.Encode()
}

// completeAuth exchanges the authorization code for an access token.
func (v *Slack) completeAuth(code string) (*OAuthV2AccessResponse, error) {
	resp, err := v.c.R().
		SetFormData(map[string]string{
			"client_id":     v.clientId,
			"client_secret": v.clientSecret,
			"code":          code,
			"redirect_uri":  v.redirectURI,
		}).
		Post("/oauth.v2.access")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("oauth.v2.access returned status %d", resp.StatusCode())
	}

	var result OAuthV2AccessResponse
	if err := sonic.Unmarshal(resp.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("parsing oauth.v2.access response: %w", err)
	}

	if !result.OK {
		return nil, fmt.Errorf("oauth.v2.access error: %s", result.Error)
	}

	v.accessToken = result.AuthedUser.AccessToken
	return &result, nil
}

// GetAccessToken implements the OAuthProvider interface. It exchanges the
// authorization code from the callback for a user access token and returns
// the token data as a KV map compatible with the oauth table storage.
func (v *Slack) GetAccessToken(ctx fiber.Ctx) (types.KV, error) {
	code := ctx.Query("code")
	if code == "" {
		return nil, fmt.Errorf("missing authorization code")
	}

	tokenResp, err := v.completeAuth(code)
	if err != nil {
		return nil, err
	}

	extra, err := sonic.Marshal(tokenResp)
	if err != nil {
		return nil, err
	}

	return types.KV{
		"name":  ID,
		"type":  ID,
		"token": v.accessToken,
		"extra": extra,
	}, nil
}

// GetIdentity uses the access token to fetch the authenticated user's
// identity from the Slack users.identity API.
func (v *Slack) GetIdentity() (*IdentityResponse, error) {
	resp, err := v.c.R().
		SetHeader("Authorization", "Bearer "+v.accessToken).
		Get("/users.identity")
	if err != nil {
		return nil, fmt.Errorf("users.identity request failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("users.identity returned status %d", resp.StatusCode())
	}

	var result IdentityResponse
	if err := sonic.Unmarshal(resp.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("parsing users.identity response: %w", err)
	}

	if !result.OK {
		return nil, fmt.Errorf("users.identity error: %s", result.Error)
	}

	return &result, nil
}
