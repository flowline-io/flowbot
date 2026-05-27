// Package slack implements the Slack API provider.
package slack

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"
	"resty.dev/v3"

	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/utils"
)

const (
	ID              = "slack"
	ClientIdKey     = "id"
	ClientSecretKey = "secret"
)

// OAuth interface compliance check.
var _ providers.OAuthProvider = (*Slack)(nil)
var _ providers.OAuthRefresher = (*Slack)(nil)

// Register registers the Slack OAuth provider in the global provider registry.
func Register() {
	providers.RegisterOAuthProvider(ID, func() providers.OAuthProvider {
		return GetClient()
	})
}

// Slack implements the OAuthProvider interface for Slack's OAuth v2 flow
// using the Sign in with Slack (identity.*) scopes.
type Slack struct {
	c            *resty.Client
	clientId     string
	clientSecret string
	redirectURI  string
	accessToken  string
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

// GetClient reads OAuth provider config and returns a new Slack client
// suitable for OAuth authorization flows.
func GetClient() *Slack {
	id, _ := providers.GetConfig(ID, ClientIdKey)
	secret, _ := providers.GetConfig(ID, ClientSecretKey)
	return NewSlack(id.String(), secret.String(), "", "")
}

// GetAuthorizeURL returns the Slack OAuth v2 authorization URL with
// identity.basic and identity.avatar user scopes. The state parameter
// is included for CSRF protection.
func (v *Slack) GetAuthorizeURL(state string) string {
	redirectURI := providers.RedirectURI(ID, state)
	params := url.Values{}
	params.Set("client_id", v.clientId)
	params.Set("user_scope", "identity.basic,identity.avatar")
	params.Set("redirect_uri", redirectURI)
	if state != "" {
		params.Set("state", state)
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
// the token data as a typed OAuthToken.
func (v *Slack) GetAccessToken(ctx fiber.Ctx) (*providers.OAuthToken, error) {
	v.redirectURI = providers.RedirectURI(ID, ctx.Params("flag"))
	code := ctx.Query("code")
	if code == "" {
		return nil, fmt.Errorf("missing authorization code")
	}

	tokenResp, err := v.completeAuth(code)
	if err != nil {
		return nil, err
	}

	return &providers.OAuthToken{
		Name:        ID,
		Type:        ID,
		AccessToken: v.accessToken,
		TokenType:   tokenResp.AuthedUser.TokenType,
		Scope:       tokenResp.AuthedUser.Scope,
		Extra:       tokenResp,
	}, nil
}

// RefreshAccessToken implements OAuthRefresher for Slack.
// Slack supports token rotation for certain scopes via the refresh_token grant.
func (v *Slack) RefreshAccessToken(ctx context.Context, refreshToken string) (*providers.OAuthToken, error) {
	resp, err := v.c.R().
		SetContext(ctx).
		SetFormData(map[string]string{
			"client_id":     v.clientId,
			"client_secret": v.clientSecret,
			"grant_type":    "refresh_token",
			"refresh_token": refreshToken,
		}).
		Post("/oauth.v2.access")
	if err != nil {
		return nil, fmt.Errorf("slack refresh token: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("slack refresh token: status %d", resp.StatusCode())
	}

	var result OAuthV2AccessResponse
	if err := sonic.Unmarshal(resp.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("slack refresh token parse: %w", err)
	}

	if !result.OK {
		return nil, fmt.Errorf("slack refresh token error: %s", result.Error)
	}

	return &providers.OAuthToken{
		Name:         ID,
		Type:         ID,
		AccessToken:  result.AuthedUser.AccessToken,
		RefreshToken: result.AuthedUser.RefreshToken,
		TokenType:    result.AuthedUser.TokenType,
		Scope:        result.AuthedUser.Scope,
		Extra:        &result,
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
