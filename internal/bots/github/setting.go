package github

import (
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/ruleset/setting"
)

const (
	repoSettingKey = "repo"
)

var settingRules = setting.Rule([]setting.Row{
	{Key: repoSettingKey, Type: types.FormFieldText, Title: "Repo"},
})
