package providers

import (
	"encoding/json"
	"fmt"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/gofiber/fiber/v3"
	"github.com/tidwall/gjson"
)

type OAuthProvider interface {
	GetAuthorizeURL() string
	GetAccessToken(ctx fiber.Ctx) (types.KV, error)
}

func RedirectURI(name string, flag string) string {
	return fmt.Sprintf("%s/oauth/%s/%s", types.AppUrl(), name, flag)
}

var Configs json.RawMessage

var ErrMissingConfig = fmt.Errorf("provider configs are empty")

func GetConfig(name, key string) (gjson.Result, error) {
	if len(Configs) == 0 {
		return gjson.Result{}, ErrMissingConfig
	}
	value := gjson.GetBytes(Configs, fmt.Sprintf("%s.%s", name, key))
	return value, nil
}
