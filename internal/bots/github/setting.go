package github

import (
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/setting"
)

const (
	repoSettingKey = "repo"
)

var settingRules = setting.Rule([]setting.Row{
	{Key: repoSettingKey, Type: types.FormFieldText, Title: "Repo"},
})
