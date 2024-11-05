package cloudflare

import (
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/ruleset/setting"
)

const (
	tokenSettingKey     = "token"
	zoneIdSettingKey    = "zone_id"
	accountIdSettingKey = "account_id"
)

var settingRules = setting.Rule([]setting.Row{
	{Key: tokenSettingKey, Type: types.FormFieldText, Title: "Token"},
	{Key: zoneIdSettingKey, Type: types.FormFieldText, Title: "Zone Id"},
	{Key: accountIdSettingKey, Type: types.FormFieldText, Title: "Account Id"},
})
