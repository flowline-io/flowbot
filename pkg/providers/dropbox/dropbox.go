package dropbox

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/providers"
	"github.com/go-resty/resty/v2"
	"github.com/gofiber/fiber/v2"
	jsoniter "github.com/json-iterator/go"
)

const (
	ID              = "dropbox"
	ClientIdKey     = "key"
	ClientSecretKey = "secret"
)

type Dropbox struct {
	c            *resty.Client
	clientId     string
	clientSecret string
	redirectURI  string
	accessToken  string
}

func NewDropbox(clientId, clientSecret, redirectURI, accessToken string) *Dropbox {
	v := &Dropbox{clientId: clientId, clientSecret: clientSecret, redirectURI: redirectURI, accessToken: accessToken}

	v.c = resty.New()
	v.c.SetBaseURL("https://api.dropboxapi.com")
	v.c.SetTimeout(time.Minute)

	return v
}

func (v *Dropbox) GetAuthorizeURL() string {
	return fmt.Sprintf("https://www.dropbox.com/oauth2/authorize?client_id=%s&response_type=code&redirect_uri=%s", v.clientId, v.redirectURI)
}

func (v *Dropbox) completeAuth(code string) (interface{}, error) {
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
		var json = jsoniter.ConfigCompatibleWithStandardLibrary
		err = json.Unmarshal(resp.Body(), &result)
		if err != nil {
			return nil, err
		}
		v.accessToken = result.AccessToken
		return &result, nil
	} else {
		return nil, fmt.Errorf("%d, %s", resp.StatusCode(), string(resp.Body()))
	}
}

func (v *Dropbox) Redirect(req *http.Request) (string, error) {
	appRedirectURI := v.GetAuthorizeURL()
	return appRedirectURI, nil
}

func (v *Dropbox) GetAccessToken(ctx *fiber.Ctx) (types.KV, error) {
	code := ctx.Query("code")
	clientId, _ := providers.GetConfig(ID, ClientIdKey)
	clientSecret, _ := providers.GetConfig(ID, ClientSecretKey)
	v.clientId = clientId.String()
	v.clientSecret = clientSecret.String()

	tokenResp, err := v.completeAuth(code)
	if err != nil {
		return nil, err
	}

	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	extra, err := json.Marshal(&tokenResp)
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

func (v *Dropbox) Upload(path string, content io.Reader) error {
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	apiArg, err := json.Marshal(map[string]interface{}{
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
		SetContentLength(true).
		SetBody(content).
		Post("https://content.dropboxapi.com/2/files/upload")
	if err != nil {
		return err
	}

	if resp.StatusCode() == http.StatusOK {
		return nil
	} else {
		return fmt.Errorf("%d, %s", resp.StatusCode(), string(resp.Body()))
	}
}
