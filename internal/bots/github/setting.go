package github

import (
	"github.com/sysatom/flowbot/internal/ruleset/setting"
	"github.com/sysatom/flowbot/internal/types"
)

const (
	repoSettingKey = "repo"
)

var settingRules = setting.Rule([]setting.Row{
	{Key: repoSettingKey, Type: types.FormFieldText, Title: "Repo"},
})
