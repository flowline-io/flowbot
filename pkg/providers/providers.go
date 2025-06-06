package providers

import (
	"encoding/json"
	"errors"
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

func GetConfig(name, key string) (gjson.Result, error) {
	if len(Configs) == 0 {
		return gjson.Result{}, errors.New("error configs")
	}
	value := gjson.GetBytes(Configs, fmt.Sprintf("%s.%s", name, key))
	return value, nil
}
