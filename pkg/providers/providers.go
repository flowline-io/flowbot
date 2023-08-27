package providers

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/tidwall/gjson"
	"net/http"
)

type OAuthProvider interface {
	AuthorizeURL() string
	GetAccessToken(code string) (interface{}, error)
	Redirect(req *http.Request) (string, error)
	StoreAccessToken(req *http.Request) (map[string]interface{}, error)
}

func RedirectURI(category string, uid1, uid2 types.Uid) string {
	return fmt.Sprintf("%s/extra/oauth/%s/%d/%d", types.AppUrl(), category, uid1, uid2)
}

var Configs json.RawMessage

func GetConfig(name, key string) (gjson.Result, error) {
	if len(Configs) == 0 {
		return gjson.Result{}, errors.New("error configs")
	}
	value := gjson.GetBytes(Configs, fmt.Sprintf("%s.%s", name, key))
	return value, nil
}
