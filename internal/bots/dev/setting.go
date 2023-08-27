package dev

import (
	"github.com/sysatom/flowbot/internal/ruleset/setting"
	"github.com/sysatom/flowbot/internal/types"
)

const (
	secretSettingKey = "secret"
	numberSettingKey = "number"
)

var settingRules = setting.Rule([]setting.Row{
	{Key: secretSettingKey, Type: types.FormFieldText, Title: "Key"},
	{Key: numberSettingKey, Type: types.FormFieldNumber, Title: "Number"},
})
