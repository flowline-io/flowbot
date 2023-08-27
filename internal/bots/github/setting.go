package github

import (
	"github.com/flowline-io/flowbot/internal/ruleset/setting"
	"github.com/flowline-io/flowbot/internal/types"
)

const (
	repoSettingKey = "repo"
)

var settingRules = setting.Rule([]setting.Row{
	{Key: repoSettingKey, Type: types.FormFieldText, Title: "Repo"},
})
