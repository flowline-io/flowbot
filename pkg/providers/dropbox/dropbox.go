// Package dropbox implements the Dropbox provider.
package dropbox

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"
	"resty.dev/v3"

	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/flowline-io/flowbot/pkg/utils"
)

const (
	ID              = "dropbox"
	ClientIdKey     = "key"
	ClientSecretKey = "secret"
)

// OAuth interface compliance check.
var _ providers.OAuthProvider = (*Dropbox)(nil)
var _ providers.OAuthRefresher = (*Dropbox)(nil)

func init() {
	providers.RegisterOAuthProvider(ID, func() providers.OAuthProvider {
		return GetClient()
	})
}

type Dropbox struct {
	c            *resty.Client
	clientId     string
	clientSecret string
	redirectURI  string
	accessToken  string
}

func NewDropbox(clientId, clientSecret, redirectURI, accessToken string) *Dropbox {
	v := &Dropbox{clientId: clientId, clientSecret: clientSecret, redirectURI: redirectURI, accessToken: accessToken}

	v.c = utils.DefaultRestyClient()
	v.c.SetBaseURL("https://api.dropboxapi.com")

	return v
}

// GetClient reads OAuth provider config and returns a new Dropbox client
// suitable for OAuth authorization flows.
func GetClient() *Dropbox {
	id, _ := providers.GetConfig(ID, ClientIdKey)
	secret, _ := providers.GetConfig(ID, ClientSecretKey)
	return NewDropbox(id.String(), secret.String(), "", "")
}

func (v *Dropbox) GetAuthorizeURL(state string) string {
	redirectURI := providers.RedirectURI(ID, state)
	return fmt.Sprintf(
		"https://www.dropbox.com/oauth2/authorize?client_id=%s&response_type=code&redirect_uri=%s&state=%s",
		v.clientId, redirectURI, state,
	)
}

func (v *Dropbox) completeAuth(code string) (*TokenResponse, error) {
	resp, err := v.c.R().
		SetBasicAuth(v.clientId, v.clientSecret).
		SetFormData(map[string]string{
			"code":         code,
			"grant_type":   "authorization_code",
			"redirect_uri": v.redirectURI,
		}).
		Post("/oauth2/token")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		var result TokenResponse
		err = sonic.Unmarshal(resp.Bytes(), &result)
		if err != nil {
			return nil, err
		}
		v.accessToken = result.AccessToken
		return &result, nil
	}
	return nil, fmt.Errorf("%d, %s", resp.StatusCode(), utils.BytesToString(resp.Bytes()))
}

func (v *Dropbox) GetAccessToken(ctx fiber.Ctx) (*providers.OAuthToken, error) {
	v.redirectURI = providers.RedirectURI(ID, ctx.Params("flag"))
	code := ctx.Query("code")

	tokenResp, err := v.completeAuth(code)
	if err != nil {
		return nil, err
	}

	var expiresAt *time.Time
	if tokenResp.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		expiresAt = &t
	}

	return &providers.OAuthToken{
		Name:         ID,
		Type:         ID,
		AccessToken:  v.accessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    expiresAt,
		TokenType:    tokenResp.TokenType,
		Scope:        tokenResp.Scope,
		Extra:        tokenResp,
	}, nil
}

// RefreshAccessToken implements OAuthRefresher for Dropbox.
// Dropbox supports token refresh via the refresh_token grant.
func (v *Dropbox) RefreshAccessToken(ctx context.Context, refreshToken string) (*providers.OAuthToken, error) {
	resp, err := v.c.R().
		SetContext(ctx).
		SetBasicAuth(v.clientId, v.clientSecret).
		SetFormData(map[string]string{
			"grant_type":    "refresh_token",
			"refresh_token": refreshToken,
		}).
		Post("/oauth2/token")
	if err != nil {
		return nil, fmt.Errorf("dropbox refresh token: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("dropbox refresh token: %d, %s", resp.StatusCode(), utils.BytesToString(resp.Bytes()))
	}

	var result TokenResponse
	if err := sonic.Unmarshal(resp.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("dropbox refresh token parse: %w", err)
	}

	var expiresAt *time.Time
	if result.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
		expiresAt = &t
	}

	return &providers.OAuthToken{
		Name:         ID,
		Type:         ID,
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    expiresAt,
		TokenType:    result.TokenType,
		Scope:        result.Scope,
		Extra:        &result,
	}, nil
}

func (v *Dropbox) Upload(path string, content io.Reader) error {
	apiArg, err := sonic.Marshal(map[string]any{
		"path":            path,
		"mode":            "add",
		"autorename":      true,
		"mute":            false,
		"strict_conflict": false,
	})
	if err != nil {
		return err
	}
	resp, err := v.c.R().
		SetAuthToken(v.accessToken).
		SetHeader("Content-Type", "application/octet-stream").
		SetHeader("Dropbox-API-Arg", string(apiArg)).
		SetBody(content).
		Post("https://content.dropboxapi.com/2/files/upload")
	if err != nil {
		return err
	}

	if resp.StatusCode() == http.StatusOK {
		return nil
	}
	return fmt.Errorf("%d, %s", resp.StatusCode(), utils.BytesToString(resp.Bytes()))
}
