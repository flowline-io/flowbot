package dev

import (
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/setting"
)

const (
	secretSettingKey = "secret"
	numberSettingKey = "number"
)

var settingRules = setting.Rule([]setting.Row{
	{Key: secretSettingKey, Type: types.FormFieldText, Title: "Key"},
	{Key: numberSettingKey, Type: types.FormFieldNumber, Title: "Number"},
})
