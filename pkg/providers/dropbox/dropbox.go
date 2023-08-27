package dropbox

import (
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"io"
	"net/http"
	"time"
)

const (
	ID              = "dropbox"
	ClientIdKey     = "key"
	ClientSecretKey = "secret"
)

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	UID         string `json:"uid"`
	AccountID   string `json:"account_id"`
	Scope       string `json:"scope"`
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

	v.c = resty.New()
	v.c.SetBaseURL("https://api.dropboxapi.com")
	v.c.SetTimeout(time.Minute)

	return v
}

func (v *Dropbox) AuthorizeURL() string {
	return fmt.Sprintf("https://www.dropbox.com/oauth2/authorize?client_id=%s&response_type=code&redirect_uri=%s", v.clientId, v.redirectURI)
}

func (v *Dropbox) GetAccessToken(code string) (interface{}, error) {
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
	clientId := "" // todo
	v.clientId = clientId

	appRedirectURI := v.AuthorizeURL()
	return appRedirectURI, nil
}

func (v *Dropbox) StoreAccessToken(req *http.Request) (map[string]interface{}, error) {
	code := req.URL.Query().Get("code")
	clientId := ""     // todo
	clientSecret := "" // todo
	v.clientId = clientId
	v.clientSecret = clientSecret

	tokenResp, err := v.GetAccessToken(code)
	if err != nil {
		return nil, err
	}

	extra, err := json.Marshal(&tokenResp)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"name":  ID,
		"type":  ID,
		"token": v.accessToken,
		"extra": extra,
	}, nil
}

func (v *Dropbox) Upload(path string, content io.Reader) error {
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